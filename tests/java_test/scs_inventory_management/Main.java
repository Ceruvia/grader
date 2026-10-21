import java.util.*;

public class Main {
    public static void main(String[] args) {
        Scanner input = new Scanner(System.in);
        int caseType = input.nextInt();
        switch (caseType) {
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
        input.close();
    }

    private static void tc1() {
        Baju baju1 = new Baju("Kaos", "White", 50000);
        Baju baju2 = new Baju("Tank Top", "Black", 10000);
        Handphone handphone1 = new Handphone("HP2030", "Oppo", 1000000);
        Handphone handphone2 = new Handphone("HP2256", "Xiaomi", 2000000);

        InventoryManager<Barang> manager = new InventoryManager<>();
        System.out.println("ADD ITEMS");
        manager.addCatalog(baju1, 10);
        manager.addCatalog(baju2, 5);
        manager.addCatalog(handphone1, 20);
        manager.addCatalog(handphone2, 15);
        manager.printInventory();
        System.out.println("REMOVE ITEMS");
        manager.removeCatalog();
        manager.removeCatalog();
        manager.printInventory();
        System.out.println("EMPTY ARRAY");
        manager.removeCatalog();
        manager.removeCatalog();
        manager.printInventory();
        manager.removeCatalog();
    }

    private static void tc2() {
        Baju baju1 = new Baju("Kaos", "White", 50);
        Baju baju2 = new Baju("Tank Top", "Black", 10);
        Handphone handphone1 = new Handphone("HP2030", "Oppo", 1000);
        Handphone handphone2 = new Handphone("HP2256", "Xiaomi", 2000);

        InventoryManager<Barang> manager = new InventoryManager<>();
        manager.addCatalog(baju1, 10);
        manager.addCatalog(baju2, 5);
        manager.addCatalog(handphone1, 20);
        manager.addCatalog(handphone2, 15);
        System.out.println(manager.getTotalPrice());
    }

    private static void tc3() {
        Baju baju1 = new Baju("Kaos", "White", 50000);
        Baju baju2 = new Baju("Tank Top", "Black", 10000);
        Handphone handphone1 = new Handphone("HP2030", "Oppo", 1000000);
        Handphone handphone2 = new Handphone("HP2256", "Xiaomi", 2000000);
        Handphone handphone3 = new Handphone("HP8345", "Iphone", 500000);

        InventoryManager<Barang> manager = new InventoryManager<>();
        manager.addCatalog(baju1, 10);
        manager.addCatalog(baju2, 5);
        manager.addCatalog(handphone1, 20);
        manager.addCatalog(handphone2, 15);
        manager.addCatalog(handphone3, 5);
        manager.printTotalType();
    }

    private static void tc4() {
        Baju baju1 = new Baju("Kaos", "White", 50000);
        Baju baju2 = new Baju("Tank Top", "Black", 10000);
        Handphone handphone1 = new Handphone("HP2030", "Oppo", 1000000);
        Handphone handphone2 = new Handphone("HP2256", "Xiaomi", 2000000);
        Handphone handphone3 = new Handphone("HP8345", "Iphone", 500000);

        InventoryManager<Barang> manager = new InventoryManager<>();
        manager.addCatalog(baju1, 10);
        manager.addCatalog(baju2, 5);
        manager.addCatalog(handphone1, 20);
        manager.addCatalog(handphone2, 15);
        manager.addCatalog(handphone3, 5);

        System.out.println(manager.getTotalAmount("Baju"));
        System.out.println(manager.getTotalAmount("Handphone"));
        System.out.println(manager.getTotalAmount("RANDOM"));
    }

    private static void tc5() {
        InventoryManager<Barang> manager = new InventoryManager<>();
        System.out.println(manager.getTotalAmount("Baju"));
        System.out.println(manager.getTotalPrice());
        manager.printTotalType();
        manager.printInventory();
    }

}