#include <stdio.h>
#include "array.h"

int main() {
    int n, odd;
    scanf("%d %d", &odd, &n);
    
    TabInt fib;
    MakeEmpty(&fib);
    
    long long sum = 0;
    
    SetEl(&fib, 0, 0);
    if (n > 0) {
        SetEl(&fib, 1, 1);
        if (odd) sum += GetElmt(fib, 1);
        for (int i = 2; i <= n; i++) {
            SetEl(&fib, i, GetElmt(fib, i-1) + GetElmt(fib, i-2));
            if ((odd && GetElmt(fib, i) % 2 != 0) || (!odd && GetElmt(fib, i) % 2 == 0)) {
                sum += GetElmt(fib, i);
            }
        }
    }
    
    printf("%lld\n", sum);
    
    printf("[");
    for (int i = 0; i <= NbElmt(fib); i++) {
        printf("%d", GetElmt(fib, i));
        if (i <= NbElmt(fib) - 1) {
            printf(",");
        }
    }
    printf("]\n");
    
    return 0;
}