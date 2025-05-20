#include<stdio.h>

int main () { 
    int kH, kM, kB;
    scanf("%d %d %d", &kH, &kM, &kB);
    char warna = 'R'; 
    char masuk;
    int masuk2;
    int H = 0;
    int M = 0;
    int B = 0;


    while (warna != 'E') { 
        scanf("%c", &masuk);
        warna = masuk;
        if (warna == 'E'){
            break;
        }
        scanf("%d", &masuk2);
        if (warna == 'H'){
            H += masuk2;
        }
        else if (warna == 'M'){
            M += masuk2;
        }
        else if (warna == 'B'){
            B += masuk2;
        }
    }

    printf("%d %d %d\n", H,M,B );
    if (kH < H || kM < M || kB < B) {
        printf("Tidak\n");
    }
    else if (H == 0 && M == 0 && B == 0){
        printf("Tidak\n");
    }
    else {
        printf("Ya\n");
    }
}