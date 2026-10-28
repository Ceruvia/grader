package sandboxes_test

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

// containerIPv4 returns the first non-loopback IPv4 address of the grader container.
func containerIPv4(t *testing.T) string {
	t.Helper()
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			return ipNet.IP.String()
		}
	}
	t.Fatal("no non-loopback IPv4 address")
	return ""
}

func TestSandboxDoesNotExposeSecrets(t *testing.T) {
	t.Setenv("BACKEND_API_TOKEN", "secret-canary")
	const envPath = "/run/grader/env"
	if err := os.MkdirAll(filepath.Dir(envPath), 0700); err != nil {
		t.Fatal(err)
	}
	original, originalErr := os.ReadFile(envPath)
	if originalErr != nil && !os.IsNotExist(originalErr) {
		t.Fatal(originalErr)
	}
	t.Cleanup(func() {
		if originalErr == nil {
			if err := os.WriteFile(envPath, original, 0600); err != nil {
				t.Error(err)
			}
		} else if err := os.Remove(envPath); err != nil {
			t.Error(err)
		}
	})
	if err := os.WriteFile(envPath, []byte("file-secret-canary\n"), 0600); err != nil {
		t.Fatal(err)
	}

	// A host task dir with an expected output, like the one A1 keeps outside the box.
	taskDir := filepath.Join("/var/lib/grader/tasks", "a2-probe")
	if err := os.MkdirAll(taskDir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(taskDir) })
	if err := os.WriteFile(filepath.Join(taskDir, "1.out"), []byte("task-dir-canary\n"), 0600); err != nil {
		t.Fatal(err)
	}

	// A listener the grader process can reach. The box must not reach it.
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	ip := containerIPv4(t)
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	if conn, err := net.Dial("tcp", net.JoinHostPort(ip, port)); err != nil {
		t.Fatalf("listener not reachable from the grader process, so the network check proves nothing: %v", err)
	} else {
		_ = conn.Close()
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "probe.c")
	program := `#include <arpa/inet.h>
#include <dirent.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <unistd.h>
static const char *reach(const char *ip, int port) {
    int fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0) return "<blocked>";
    struct sockaddr_in addr = {0};
    addr.sin_family = AF_INET;
    addr.sin_port = htons(port);
    inet_pton(AF_INET, ip, &addr.sin_addr);
    int ok = connect(fd, (struct sockaddr *)&addr, sizeof addr) == 0;
    close(fd);
    return ok ? "<open>" : "<blocked>";
}
int main(int argc, char **argv) {
    if (argc != 3) return 3;
    const char *token = getenv("BACKEND_API_TOKEN");
    printf("env:%s\n", token ? token : "<unset>");
    FILE *f = fopen("/run/grader/env", "r");
    if (f) {
        char line[256];
        while (fgets(line, sizeof line, f)) fputs(line, stdout);
        fclose(f);
    } else puts("file:<blocked>");
    f = fopen("/etc/passwd", "r");
    puts(f ? "etc:<readable>" : "etc:<blocked>");
    if (f) fclose(f);
    char environ1[65536] = {0};
    f = fopen("/proc/1/environ", "r");
    if (f) {
        size_t n = fread(environ1, 1, sizeof environ1 - 1, f);
        fclose(f);
        for (size_t i = 0; i < n; i++) if (!environ1[i]) environ1[i] = '\n';
    }
    puts(strstr(environ1, "secret-canary") ? "proc1:<leak>" : "proc1:<clean>");
    DIR *d = opendir("/var/lib/grader/tasks");
    f = fopen("/var/lib/grader/tasks/a2-probe/1.out", "r");
    puts(d || f ? "tasks:<readable>" : "tasks:<blocked>");
    if (d) closedir(d);
    if (f) fclose(f);
    int port = atoi(argv[2]);
    printf("net-ip:%s\n", reach(argv[1], port));
    printf("net-lo:%s\n", reach("127.0.0.1", port));
    return 0;
}
`
	if err := os.WriteFile(source, []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "probe")
	if output, err := exec.Command("gcc", "-o", binary, source).CombinedOutput(); err != nil {
		t.Fatalf("build probe: %v\n%s", err, output)
	}
	sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 500)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sbx.Cleanup() })
	sbx.SetTimeLimitInMiliseconds(1000)
	sbx.SetWallTimeLimitInMiliseconds(2000)
	sbx.SetMemoryLimitInKilobytes(65536)
	if err := sbx.AddFile(binary); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(sbx.GetBoxdir(), "probe"), 0700); err != nil {
		t.Fatal(err)
	}
	red := sandboxes.CreateRedirectionFiles(sbx)
	if err := red.CreateNewMetaFileAndRedirect("probe.meta"); err != nil {
		t.Fatal(err)
	}
	if err := red.CreateNewStandardOutputFileAndRedirect("probe.out"); err != nil {
		t.Fatal(err)
	}
	if err := red.RedirectStandardError("probe.out"); err != nil {
		t.Fatal(err)
	}
	result := sbx.Execute(*command.GetCommandBuilder("./probe").AddArgs(ip, port), red)
	output, err := os.ReadFile(filepath.Join(sbx.GetBoxdir(), "probe.out"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != sandboxes.ZERO_EXIT_CODE {
		t.Fatalf("probe failed: %+v, output %q", result, output)
	}
	want := "env:<unset>\nfile:<blocked>\netc:<blocked>\nproc1:<clean>\ntasks:<blocked>\nnet-ip:<blocked>\nnet-lo:<blocked>\n"
	if strings.Contains(string(output), "canary") || string(output) != want {
		t.Fatalf("sandbox exposes host state: %q", output)
	}
}
