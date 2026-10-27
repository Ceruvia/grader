package sandboxes_test

import (
	"encoding/json"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func TestSandboxDefaultWritableMountInventory(t *testing.T) {
	sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 838)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sbx.Cleanup(); err != nil {
			t.Error(err)
		}
	})
	sbx.SetWallTimeLimitInMiliseconds(10000)
	red := sandboxes.CreateRedirectionFiles(sbx)
	if err := red.CreateNewMetaFileAndRedirect("inventory.meta"); err != nil {
		t.Fatal(err)
	}
	if err := red.CreateNewStandardOutputFileAndRedirect("inventory.out"); err != nil {
		t.Fatal(err)
	}
	script := `import errno, json, os
mounts = []
paths = {'/', '/box', '/tmp', '/var', '/var/tmp', '/usr/tmp', '/run', '/run/user', '/root', '/home', '/dev', '/dev/shm', '/dev/mqueue', '/dev/pts', '/proc/self/fd'}
with open('/proc/self/mountinfo') as source:
 for line in source:
  left, right = line.strip().split(' - ', 1)
  fields, fs = left.split(), right.split()
  path = fields[4]
  mounts.append({'path': path, 'options': fields[5], 'filesystem': fs[0]})
  paths.add(path)
probes = []
for path in sorted(paths):
 probe = {'path': path, 'writable': False, 'device': 0, 'errno': 0}
 target = os.path.join(path, 'ceruvia-storage-probe')
 try:
  fd = os.open(target, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
  try:
   os.write(fd, b'x')
   probe['device'] = os.fstat(fd).st_dev
   probe['writable'] = True
  finally:
   os.close(fd)
   os.unlink(target)
 except OSError as exc:
  probe['errno'] = exc.errno
 probes.append(probe)
print(json.dumps({'box_device': os.stat('/box').st_dev, 'mounts': mounts, 'probes': probes}))
`
	got := sbx.Execute(*command.GetCommandBuilder("/usr/bin/python3").AddArgs("-c", script), red)
	if got.Status != sandboxes.ZERO_EXIT_CODE {
		t.Fatalf("inventory failed: %+v", got)
	}
	data, err := sandboxes.ReadBoxFile(sbx, "inventory.out", 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("real in-box mount/write inventory: %s", data)
	var inventory struct {
		BoxDevice uint64 `json:"box_device"`
		Probes    []struct {
			Path     string `json:"path"`
			Writable bool   `json:"writable"`
			Device   uint64 `json:"device"`
			Errno    int    `json:"errno"`
		} `json:"probes"`
	}
	if err := json.Unmarshal(data, &inventory); err != nil {
		t.Fatal(err)
	}
	writable := 0
	for _, probe := range inventory.Probes {
		if probe.Writable {
			writable++
			if probe.Device != inventory.BoxDevice {
				t.Errorf("unbounded writable storage outside the box filesystem: %s", probe.Path)
			}
		} else if probe.Path == "/box" || probe.Path == "/tmp" {
			t.Errorf("required scratch path is unavailable: %s errno %d", probe.Path, probe.Errno)
		}
	}
	if writable < 2 {
		t.Fatal("inventory did not exercise both scratch views")
	}
}

func TestSandboxCannotBypassInodesThroughDeviceSharedMemory(t *testing.T) {
	sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 839)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sbx.Cleanup(); err != nil {
			t.Error(err)
		}
	})
	sbx.SetWallTimeLimitInMiliseconds(10000)
	red := sandboxes.CreateRedirectionFiles(sbx)
	if err := red.CreateNewMetaFileAndRedirect("shm.meta"); err != nil {
		t.Fatal(err)
	}
	script := `import errno, os, shutil
path = '/dev/shm/ceruvia-inode-probe'
try:
 os.mkdir(path)
 for i in range(10001):
  open(path + '/' + str(i), 'w').close()
except OSError as exc:
 if exc.errno in (errno.ENOENT, errno.EACCES, errno.EPERM, errno.EROFS): raise SystemExit(0)
 if exc.errno == errno.ENOSPC: raise SystemExit(42)
 raise
else:
 raise SystemExit(43)
finally:
 if os.path.isdir(path): shutil.rmtree(path)
`
	got := sbx.Execute(*command.GetCommandBuilder("/usr/bin/python3").AddArgs("-c", script), red)
	if got.Status != sandboxes.ZERO_EXIT_CODE && !(got.Status == sandboxes.NONZERO_EXIT_CODE && got.ExitCode == 42) {
		t.Fatalf("/dev/shm must be unavailable or hit the inode bound before 10001 files, got %+v", got)
	}
}
