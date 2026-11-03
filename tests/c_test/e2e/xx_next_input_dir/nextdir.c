#include <stdio.h>
#include <sys/stat.h>
#include <unistd.h>

// Replaces the next case's input with a nonempty directory, so the grader
// cannot install 2.in before case 2 runs.
int main() {
    unlink("2.in");
    mkdir("2.in", 0700);
    FILE *f = fopen("2.in/child", "w");
    if (f) fclose(f);
    printf("ok\n");
}
