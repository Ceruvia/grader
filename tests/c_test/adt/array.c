#include "array.h"
#include <stdio.h>

/* ********** KONSTRUKTOR ********** */
/* Konstruktor : create tabel kosong */
void MakeEmpty (TabInt *T)
/* I.S. sembarang */
/* F.S. Terbentuk tabel T kosong dengan kapasitas IdxMax-IdxMin+1 */
{
	(*T).Neff = 0;
}

/* ********** SELEKTOR ********** */
/* *** Banyaknya elemen *** */
int NbElmt (TabInt T)
/* Mengirimkan banyaknya elemen efektif tabel */
/* Mengirimkan nol jika tabel kosong */
{
	return T.Neff;
}

/* *** Daya tampung container *** */
int MaxNbEl (TabInt T)
/* Mengirimkan maksimum elemen yang dapat ditampung oleh tabel */
{
	return IdxMax - IdxMin + 1;
}

/* *** Selektor INDEKS *** */
IdxType GetFirstIdx (TabInt T)
/* Prekondisi : Tabel T tidak kosong */
/* Mengirimkan indeks elemen pertama */
{
	return 1;
}
IdxType GetLastIdx (TabInt T)
/* Prekondisi : Tabel T tidak kosong */
/* Mengirimkan indeks elemen terakhir */
{
	return T.Neff;
}

/* *** Menghasilkan sebuah elemen *** */
ElType GetElmt (TabInt T, IdxType i)
/* Prekondisi : Tabel tidak kosong, i antara FirstIdx(T)..LastIdx(T) */
/* Mengirimkan elemen tabel yang ke-i */
{
	return T.TI[i];
}

/* *** Selektor SET : Mengubah nilai TABEL dan elemen tabel *** */
/* Untuk type private/limited private pada bahasa tertentu */
void SetTab (TabInt Tin, TabInt *Tout)
/* I.S. Tin terdefinisi, sembarang */
/* F.S. Tout berisi salinan Tin */
/* Assignment THsl -> Tin */
{
	(*Tout).Neff = Tin.Neff;
	for(int i = IdxMin; i <= Tin.Neff; i++){
		(*Tout).TI[i] = Tin.TI[i];
	}
}

void SetEl (TabInt *T, IdxType i, ElType v)
/* I.S. T terdefinisi, sembarang */
/* F.S. Elemen T yang ke-i bernilai v */
/* Mengeset nilai elemen tabel yang ke-i sehingga bernilai v */
{
	(*T).TI[i] = v;
	if ((*T).Neff < i){
		(*T).Neff = i;
	}
}

void SetNeff (TabInt *T, IdxType N)
/* I.S. T terdefinisi, sembarang */
/* F.S. Nilai indeks efektif T bernilai N */
/* Mengeset nilai indeks elemen efektif sehingga bernilai N */
{
	(*T).Neff = N;
}

/* ********** Test Indeks yang valid ********** */
boolean IsIdxValid (TabInt T, IdxType i)
/* Prekondisi : i sembarang */
/* Mengirimkan true jika i adalah indeks yang valid utk ukuran tabel */
/* yaitu antara indeks yang terdefinisi utk container*/
{
	if ((i >= IdxMin) && (i <= IdxMax)){
		return true;
	} else {
		return false;
	}
}

boolean IsIdxEff (TabInt T, IdxType i)
/* Prekondisi : i sembarang*/
/* Mengirimkan true jika i adalah indeks yang terdefinisi utk tabel */
/* yaitu antara FirstIdx(T)..LastIdx(T) */
{
	if ((i >= IdxMin) && (i <= T.Neff)){
		return true;
	} else {
		return false;
	}
}

/* ********** TEST KOSONG/PENUH ********** */
/* *** Test tabel kosong *** */
boolean IsEmpty (TabInt T)
/* Mengirimkan true jika tabel T kosong, mengirimkan false jika tidak */
{
	if (T.Neff > 0){
		return false;
	} else {
		return true;
	}
}
/* *** Test tabel penuh *** */
boolean IsFull (TabInt T)
/* Mengirimkan true jika tabel T penuh, mengirimkan false jika tidak */
{
	if (T.Neff == IdxMax){
		return true;
	} else {
		return false;
	}
}

/* ********** BACA dan TULIS dengan INPUT/OUTPUT device ********** */
void TulisIsi (TabInt T)
/* Proses : Menuliskan isi tabel dengan traversal */
/* I.S. T boleh kosong */
/* F.S. Jika T tidak kosong : indeks dan elemen tabel ditulis berderet ke bawah */
/* Jika isi tabel [1,2,3] maka akan diprint
0:1
1:2
2:3
*/
/* Jika T kosong : Hanya menulis "Tabel kosong" */
{
	int i;
	if (IsEmpty(T)) {
		printf("Tabel kosong\n");
	} else {
		for (i = IdxMin; i <= T.Neff; i++) {
			printf("%d:%d\n", i, T.TI[i]);
		}
	}
}

/* ********** OPERATOR ARITMATIKA ********** */
/* *** Aritmatika tabel : Penjumlahan, pengurangan, perkalian, ... *** */
TabInt PlusTab (TabInt T1, TabInt T2)
/* Prekondisi : T1 dan T2 berukuran sama dan tidak kosong */
/* Mengirimkan T1 + T2 */
{
	TabInt THasil;
	int i;

	THasil.Neff = T1.Neff;
	for (i = IdxMin; i <= T1.Neff; i++) {
		THasil.TI[i] = T1.TI[i] + T2.TI[i];
	}
	return THasil;
}
TabInt MinusTab (TabInt T1, TabInt T2)
/* Prekondisi : T1 dan T2 berukuran sama dan tidak kosong */
/* Mengirimkan T1 - T2 */
{
	TabInt THasil;
	int i;

	THasil.Neff = T1.Neff;
	for (i = IdxMin; i <= T1.Neff; i++) {
		THasil.TI[i] = T1.TI[i] - T2.TI[i];
	}
	return THasil;
}
/* ********** NILAI EKSTREM ********** */
ElType ValMax (TabInt T)
/* Prekondisi : Tabel T tidak kosong */
/* Mengirimkan nilai maksimum tabel */
{
	int i;
	ElType maxVal = T.TI[IdxMin];
	for (i = IdxMin + 1; i <= T.Neff; i++) {
		if (maxVal < T.TI[i]) {
			maxVal = T.TI[i];
		}
	}
	return maxVal;
}

ElType ValMin (TabInt T)
/* Prekondisi : Tabel T tidak kosong */
/* Mengirimkan nilai minimum tabel */
{
	int i;
	ElType minVal = T.TI[IdxMin];
	for (i = IdxMin + 1; i <= T.Neff; i++) {
		if (minVal > T.TI[i]) {
			minVal = T.TI[i];
		}
	}
	return minVal;
}

/* *** Mengirimkan indeks elemen bernilai ekstrem *** */
IdxType IdxMaxTab (TabInt T)
/* Prekondisi : Tabel T tidak kosong */
/* Mengirimkan indeks i dengan elemen ke-i adalah nilai maksimum pada tabel */
{
	int i;
	ElType maxVal = T.TI[IdxMin];
	IdxType idx = IdxMin;
	for (i = IdxMin + 1; i <= T.Neff; i++) {
		if (maxVal < T.TI[i]) {
			maxVal = T.TI[i];
			idx = i;
		}
	}
	return idx;
}

IdxType IdxMinTab (TabInt T)
/* Prekondisi : Tabel tidak kosong */
/* Mengirimkan indeks i */
/* dengan elemen ke-i nilai minimum pada tabel */
{
	int i;
	ElType minVal = T.TI[IdxMin];
	IdxType idx = IdxMin;
	for (i = IdxMin + 1; i <= T.Neff; i++) {
		if (minVal > T.TI[i]) {
			minVal = T.TI[i];
			idx = i;
		}
	}
	return idx;
}