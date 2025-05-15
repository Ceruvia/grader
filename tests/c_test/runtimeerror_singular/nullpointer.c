#include <stdio.h>

int main() {
    int *p = NULL;
    int input; scanf("%d", &input);

    if (input == 0) {
        *p = 42;
    } else {
        printf("Gurt: Yo\n");
    }

    return 0;
}
