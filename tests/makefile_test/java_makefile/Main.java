import java.util.Scanner;

public class Main {
    public static void main(String[] args) {
        Scanner scanner = new Scanner(System.in);
        switch (scanner.nextInt()) {
            case 1:
                tc1();
                break;
            case 2:
                tc2();
                break;
            case 3:
                tc3();
                break;
            case 4:
                tc4();
                break;
            case 5:
                tc5();
                break;
        }
        scanner.close();
    }

    private static void tc1() {
        Baju b1 = new Baju("Gaun", "Merah");
        Keranjang<Baju> keranjang = new Keranjang<Baju>(b1);
        System.out.println(keranjang.getType());
    }

    private static void tc2() {
        Handphone h1 = new Handphone("Oppo", 20000);
        Keranjang<Handphone> keranjang = new Keranjang<Handphone>(h1);
        System.out.println(keranjang.getType());
    }

    private static void tc3() {
        Baju b1 = new Baju("Gaun", "Merah");
        Keranjang<Baju> keranjang = new Keranjang<Baju>(b1);
        System.out.println(keranjang.getBarang());
    }

    private static void tc4() {
        Handphone h1 = new Handphone("Oppo", 20000);
        Keranjang<Handphone> keranjang = new Keranjang<Handphone>(h1);
        System.out.println(keranjang.getBarang());
    }

    private static void tc5() {
        Baju b1 = new Baju("Gaun", "Merah");
        Handphone h1 = new Handphone("Oppo", 20000);
        Keranjang<Handphone> keranjang = new Keranjang<Handphone>(h1);
        Keranjang<Baju> keranjang2 = new Keranjang<Baju>(b1);
        keranjang.printBarang();
        keranjang2.printBarang();
    }

}