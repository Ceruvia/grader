#include <stdio.h>
#include "stack.h"
#include "boolean.h"

int main() {
    int panjang, i = 0, idx, idx2, idx3;
    infotype sampah;

    scanf("%d", &panjang);

    infotype kata[panjang];

    scanf("%s", kata);

    Stack temp;
    CreateEmpty(&temp);

    while (kata[i] != '\0') {
        idx = i + 1;
        idx2 = i + 2;
        idx3 = i + 3;

        if (kata[i] == '<') {
            if (kata[idx] == '>') {
                Push(&temp, kata[idx]);
                Push(&temp, kata[i]);
                i = idx;
            }
            else if (kata[idx] == '<') {
                if (kata[idx2] == '>') {
                    if (kata[idx3] == '>') {
                    Push(&temp, kata[idx2]);
                    Push(&temp, kata[i]);
                    Push(&temp, kata[idx3]);
                    Push(&temp, kata[idx]);
                    i = idx3;
                    }
                }
            }
        }
        
        i++;
    }

    while (!IsEmpty(temp)) {
        printf("%c", InfoTop(temp));
        Pop(&temp, &sampah);
    }
    printf("\n");

    return 0;
}