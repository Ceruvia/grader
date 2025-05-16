/* File : listsirkuler.c */
/* ADT List Sirkuler dengan elemen terakhir menunjuk pada elemen pertama   */
/* Representasi address dengan pointer */
/* infotype adalah integer */
/* Author: David / 13513019 */
/* Tanggal: 18 Oktober 2016 */

#include "listsirkuler.h"
#include <stdio.h>
#include <stdlib.h>

/* PROTOTYPE */
/****************** TEST LIST KOSONG ******************/
boolean IsEmpty (List L)
/* Mengirim true jika list kosong */
{
  return First(L) == Nil;
}

/****************** PEMBUATAN LIST KOSONG ******************/
void CreateEmpty (List *L)
/* I.S. sembarang             */
/* F.S. Terbentuk list kosong */
{
  First(*L) = Nil;
}

/****************** Manajemen Memori ******************/
address Alokasi (infotype X)
/* Mengirimkan address hasil alokasi sebuah elemen */
/* Jika alokasi berhasil, maka address tidak nil, dan misalnya */
/* menghasilkan P, maka info(P)=X, Next(P)=Nil */
/* Jika alokasi gagal, mengirimkan Nil */
{
  address P = (address) malloc (sizeof(ElmtList));
  if (P != NULL) {
    Info(P) = X;
    Next(P) = Nil;
  }

  return P;
}

void Dealokasi (address P)
/* I.S. P terdefinisi */
/* F.S. P dikembalikan ke sistem */
/* Melakukan dealokasi/pengembalian address P */
{
  free(P);
}

/****************** PENCARIAN SEBUAH ELEMEN LIST ******************/
address Search (List L, infotype X)
/* Mencari apakah ada elemen list dengan Info(P)= X */
/* Jika ada, mengirimkan address elemen tersebut. */
/* Jika tidak ada, mengirimkan Nil */
{
  address P = First(L);
  boolean found = false;
  do {
    if (Info(P) == X) {
      found = true;
    } else {
      P = Next(P);
    }
  }
  while (P != First(L) && !found);

  if (Info(P) == X) {
    return P;
  }
  return Nil;

}

/****************** PRIMITIF BERDASARKAN NILAI ******************/
/*** PENAMBAHAN ELEMEN ***/
void InsVFirst (List *L, infotype X)
/* I.S. L mungkin kosong */
/* F.S. Melakukan alokasi sebuah elemen dan */
/* menambahkan elemen pertama dengan nilai X jika alokasi berhasil */
{
  address P = Alokasi(X);
  if (P != Nil) {
    InsertFirst(L, P);
  }
}

void InsVLast (List *L, infotype X)
/* I.S. L mungkin kosong */
/* F.S. Melakukan alokasi sebuah elemen dan */
/* menambahkan elemen list di akhir: elemen terakhir yang baru */
/* bernilai X jika alokasi berhasil. Jika alokasi gagal: I.S.= F.S. */
{
  address P = Alokasi(X);
  if (P != Nil) {
    InsertLast(L, P);
  }
}

/*** PENGHAPUSAN ELEMEN ***/
void DelVFirst (List *L, infotype * X)
/* I.S. List L tidak kosong  */
/* F.S. Elemen pertama list dihapus: nilai info disimpan pada X */
/*      dan alamat elemen pertama di-dealokasi */
{
  address P;
  DelFirst(L, &P);
  *X = Info(P);
  Dealokasi(P);
}

void DelVLast (List *L, infotype * X)
/* I.S. list tidak kosong */
/* F.S. Elemen terakhir list dihapus: nilai info disimpan pada X */
/*      dan alamat elemen terakhir di-dealokasi */
{
  address P;
  DelLast(L, &P);
  *X = Info(P);
  Dealokasi(P);
}

/****************** PRIMITIF BERDASARKAN ALAMAT ******************/
/*** PENAMBAHAN ELEMEN BERDASARKAN ALAMAT ***/
void InsertFirst (List *L, address P)
/* I.S. Sembarang, P sudah dialokasi  */
/* F.S. Menambahkan elemen ber-address P sebagai elemen pertama */
{
  if (IsEmpty(*L)) {
    First(*L) = P;
    Next(P) = First(*L);
  } else {
    address last = First(*L);
    while (Next(last) != First(*L)) {
      last = Next(last);
    }
    Next(P) = First(*L);
    First(*L) = P;
    Next(last) = First(*L);
  }
}

void InsertAfter (List *L, address P, address Prec)
/* I.S. Prec pastilah elemen list dan bukan elemen terakhir, */
/*      P sudah dialokasi  */
/* F.S. Insert P sebagai elemen sesudah elemen beralamat Prec */
{
  Next(P) = Next(Prec);
  Next(Prec) = P;
}

void InsertLast (List *L, address P)
/* I.S. Sembarang, P sudah dialokasi  */
/* F.S. P ditambahkan sebagai elemen terakhir yang baru */
{
  if (IsEmpty(*L)) {
    First(*L) = P;
    Next(P) = P;
  } else {
    address last = First(*L);
    while (Next(last) != First(*L)) {
      last = Next(last);
    }
    Next(last) = P;
    Next(P) = First(*L);
  }
}

/*** PENGHAPUSAN SEBUAH ELEMEN ***/
void DelFirst (List *L, address *P)
/* I.S. List tidak kosong */
/* F.S. P adalah alamat elemen pertama list sebelum penghapusan */
/*      Elemen list berkurang satu (mungkin menjadi kosong) */
/* First element yg baru adalah suksesor elemen pertama yang lama */
{
  address last = First(*L);
  while (Next(last) != First(*L)) {
    last = Next(last);
  }
  *P = First(*L);
  if (Next(*P) != First(*L)) {
    First(*L) = Next(*P);
    Next(last) = First(*L);
  } else {
    First(*L) = Nil;
  }
}

void DelP (List *L, infotype X)
/* I.S. Sembarang */
/* F.S. Jika ada elemen list beraddress P, dengan info(P)=X  */
/* Maka P dihapus dari list dan di-dealokasi */
/* Jika tidak ada elemen list dengan info(P)=X, maka list tetap */
/* List mungkin menjadi kosong karena penghapusan */
{
  address temp;
  address P = First(*L);
  if (Info(P) == X) {
    DelFirst(L, &temp);
    Dealokasi(temp);
  } else {
    while (Next(P) != First(*L) && Info(Next(P)) != X) {
      P = Next(P);
    }
    if (Next(P) != First(*L)) {
      DelAfter(L, &temp, P);
      Dealokasi(temp);
    }
  }
}

void DelLast (List *L, address *P)
/* I.S. List tidak kosong */
/* F.S. P adalah alamat elemen terakhir list sebelum penghapusan  */
/*      Elemen list berkurang satu (mungkin menjadi kosong) */
/* Last element baru adalah predesesor elemen terakhir yg lama, */
/* jika ada */
{
  address Pprev = Nil, Pcur = First(*L);
  while (Next(Pcur) != First(*L)) {
    Pprev = Pcur;
    Pcur = Next(Pcur);
  }
  *P = Pcur;
  if (Pprev == Nil) {
    First(*L) = Nil;
  } else {
    Next(Pprev) = First(*L);
  }
}

void DelAfter (List *L, address *Pdel, address Prec)
/* I.S. List tidak kosong. Prec adalah anggota list  */
/* F.S. Menghapus Next(Prec): */
/*      Pdel adalah alamat elemen list yang dihapus  */
{
  *Pdel = Next(Prec);
  if (*Pdel != First(*L)) {
    Next(Prec) = Next(*Pdel);
  } else {
    First(*L) = Next(First(*L));
    Next(Prec) = First(*L);
  }
}

/****************** PROSES SEMUA ELEMEN LIST ******************/
void PrintInfo (List L)
/* I.S. List mungkin kosong */
/* F.S. Jika list tidak kosong, iai list dicetak ke kanan: [e1,e2,...,en] */
/* Contoh : jika ada tiga elemen bernilai 1, 20, 30 akan dicetak: [1,20,30] */
/* Jika list kosong : menulis [] */
/* Tidak ada tambahan karakter apa pun di awal, akhir, atau di tengah */
{
  printf("[");
  address P = First(L);
  while (Next(P) != First(L)) {
    printf("%d,", Info(P));
    P = Next(P);
  }
  printf("%d", Info(P));  
  printf("]");
}