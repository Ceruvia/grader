#include <stdio.h>

void recurse() {
    recurse();
}

int main() {
    recurse();
    return 0;
}
