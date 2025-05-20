import java.util.Scanner;
import java.lang.System;

/**
 * Main.java
 * 
 * @author 18223055 Muhammad Omar Berliansyah
 */

public class Main {
    /**
     * Mengecek apabila kartu memiliki nilai 10, J, Q, K, A
     * 
     * @param cards
     * @return true apabila kartu memiliki 10 sampai As, false sebaliknya
     */
    public static boolean isRoyal(String[] cards) {
        char Devon;
        for (int i = 0; i < 4; i++){
            Devon = cards[i].charAt(0);
            if (Devon != cards[i+1].charAt(0)){
                return false;
            }
        }
        
        for (int i = 0; i < 5; i++){
            if ((cards[i].charAt(1) =='J' || cards[i].charAt(1) =='Q'|| cards[i].charAt(1) =='K' || cards[i].charAt(1) =='A')){
                return true;
            }
            if (cards[i].charAt(1)  == '1' && cards[i].charAt(2) =='0'){
                return true;
            }
        }
        return false;
    }

    /**
     * Mengecek apabila kartu dapat membentuk Full House
     * 
     * @param cards
     * @return true apabila kartu dapat membentuk Full House, false sebaliknya
     */
    public static boolean isFullHouse(String[] cards) {
        char Devon = cards[0].charAt(1);
        char Adit;
        char Denis;
        boolean same = true;
        int difference = 0;
        for (int i = 1; i < 4; i++){
            if (cards[i].length() != 3){
                Denis = cards[i].charAt(1);
                Adit = cards[i+1].charAt(1);
                
                if (Devon == Denis || Adit == Denis){
                    continue;
                }
                else { 
                    difference += 1;
                }
            }
            else{
                Denis = cards[i].charAt(1);
                Adit = cards[i+1].charAt(1);
                char sopo = cards[i].charAt(2);
                char jarwo = cards[i+1].charAt(2);
                char Adel = cards[0].charAt(2);
                
                if (((Devon == Denis) && Adel== sopo )|| (Adit == Denis && sopo == jarwo)){
                    continue;
                }
                else { 
                    difference += 1;
                }
            }
        }

        return difference <= 1? true:false;
    }

    /**
     * Mengecek apabila kartu dapat membentuk Flush
     * 
     * @param cards
     * @return true apabila kartu dapat membentuk Flush, false sebaliknya
     */
    public static boolean isFlush(String[] cards) {
        char Devon;
        for (int i = 0; i < 4; i++){
            Devon = cards[i].charAt(0);
            if (Devon != cards[i+1].charAt(0)){
                return false;
            }
        }
        return true;
    }

    /**
     * Mengembalikan rangking dari set yang dimiliki dengan rangking berikut:
     * - Royal Flush: 3
     * - Full House: 2
     * - Flush: 1
     * - High Card: 0
     * 
     * @param cards
     * @return rangking
     */
    public static int getSetRanking(String[] cards) {
        if (isRoyal(cards)){
            return 3;
        }
        else if (isFullHouse(cards)){
            return 2;
        }
        else if(isFlush(cards)){
            return 1;
        }
        else  {
            return 0;
        }
    }

    public static void main(String[] args) {
        String[] cardsTuanBil = new String[5];
        String[] cardsTuanMask = new String[5];
        int rankBil = 0, rankMask = 0;
        Scanner input = new Scanner(System.in);

        /* YOUR CODE HERE */
        for (int i = 0; i < 5; i++){
            cardsTuanBil[i] = input.next();
        }

        for (int i = 0; i < 5; i++){
            cardsTuanMask[i] = input.next();
        }
        rankBil += getSetRanking(cardsTuanBil);
        rankMask += getSetRanking(cardsTuanMask);

        if (rankBil > rankMask){
            System.out.println("Tuan Bil");
            System.out.println(rankBil);
        }

        else if (rankBil < rankMask){
            System.out.println("Tuan Mask");
            System.out.println(rankMask);
        }

        else if (rankBil == rankMask){
            System.out.println("Seri");
            System.out.println(rankMask);
        }
    }
}