#include <stdio.h>
#include "listsirkuler.h"

void test1() {
    List L;
    CreateEmpty(&L);
    InsVFirst(&L, 10);
    InsVLast(&L, 20);
    InsVLast(&L, 30);
    PrintInfo(L); //- Expected output: [10,20,30]
}

void test2() {
    List L;
    CreateEmpty(&L);
    InsVFirst(&L, 10);
    InsVFirst(&L, 20);
    InsVFirst(&L, 30);
    PrintInfo(L); // Expected output: [30,20,10]
}

void test3() {
    List L;
    CreateEmpty(&L);
    InsVFirst(&L, 10);
    InsVFirst(&L, 20);
    InsVFirst(&L, 30);
    infotype X;
    DelVFirst(&L, &X);
    PrintInfo(L); // Expected output: [20,10]
}

void test4() {
    List L;
    CreateEmpty(&L);
    InsVFirst(&L, 10);
    InsVLast(&L, 20);
    InsVLast(&L, 30);
    infotype X;
    DelVLast(&L, &X);
    PrintInfo(L); // Expected output: [10,20]
}

void test5() {
    List L;
    CreateEmpty(&L);
    InsVFirst(&L, 10);
    InsVLast(&L, 20);
    InsVLast(&L, 30);
    address P = Search(L, 20);
    if (P != Nil) {
        printf("Found: %d\n", Info(P)); // Expected output: Found: 20
    } else {
        printf("Not Found\n");
    }
}

void test6() {
    List L;
    CreateEmpty(&L);
    InsVFirst(&L, 10);
    InsVLast(&L, 20);
    InsVLast(&L, 30);
    DelP(&L, 20);
    PrintInfo(L); // Expected output: [10,30]
}

void test7() {
    List L;
    CreateEmpty(&L);
    InsVFirst(&L, 10);
    InsVLast(&L, 20);
    InsVLast(&L, 30);
    address P = First(L);
    address temp;
    DelAfter(&L, &temp, First(L));
    PrintInfo(L); // Expected output: [10,30]
}

void test8() {
    List L;
    CreateEmpty(&L);
    InsVFirst(&L, 10);
    InsVLast(&L, 20);
    InsVLast(&L, 30);
    address P = First(L);
    InsertAfter(&L, Alokasi(25), P);
    PrintInfo(L); // Expected output: [10,25,20,30]
}

void test9() {
    List L;
    CreateEmpty(&L);
    InsVFirst(&L, 10);
    InsVLast(&L, 20);
    InsVLast(&L, 30);
    InsertFirst(&L, Alokasi(5));
    PrintInfo(L); // Expected output: [5,10,20,30]
}

int main() {
    int pil;
    scanf("%d", &pil);
    switch (pil) {
        case 1: test1(); break;
        case 2: test2(); break;
        case 3: test3(); break;
        case 4: test4(); break;
        case 5: test5(); break;
        case 6: test6(); break;
        case 7: test7(); break;
        case 8: test8(); break;
        case 9: test9(); break;
        default: printf("Invalid option\n");
    }
    return 0;
}
