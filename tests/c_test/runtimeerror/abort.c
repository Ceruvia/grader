#include <stdlib.h>

// abort() raises SIGABRT on every platform, unlike undefined out-of-bounds writes.
int main(void) {
    abort();
}
