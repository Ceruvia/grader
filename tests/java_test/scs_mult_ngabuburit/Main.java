import java.lang.System;
import java.util.Scanner;

public class Main {
    public static void main(String args[]) {
        Scanner scanner = new Scanner(System.in);
        int tcType = scanner.nextInt();
        switch (tcType) {
            case 1:
                tc1(scanner);
                break;
            case 2:
                tc2(scanner);
                break;
            case 3:
                tc3(scanner);
                break;
            case 4:
                tc4(scanner);
                break;
        }
    }

    private static void tc1(Scanner s) {
        LaptopCommand lc = new LaptopCommand(new Laptop());
        int len = s.nextInt();

        for (int i=0; i<len; i++) {
            int command = s.nextInt();
            switch (command) {
                case 1:
                    lc.execute();
                    break;
                case 2:
                    lc.undo();
                    break;
            }
        }

    }

    private static void tc2(Scanner s) {
        PintuCommand pc = new PintuCommand(new Pintu());
        int len = s.nextInt();

        for (int i=0; i<len; i++) {
            int command = s.nextInt();
            switch (command) {
                case 1:
                    pc.execute();
                    break;
                case 2:
                    pc.undo();
                    break;
            }
        }
    }

    private static void tc3(Scanner s) {
        RumahCommand rc = new RumahCommand(new Rumah());
        int len = s.nextInt();

        for (int i=0; i<len; i++) {
            int command = s.nextInt();
            switch (command) {
                case 1:
                    rc.execute();
                    break;
                case 2:
                    rc.undo();
                    break;
            }
        }
    }

    private static void tc4(Scanner s) {
        Ngabuburit n = new Ngabuburit(null);
        int initialCommand = s.nextInt();

        n.changeCommand(convertNumToCommand(initialCommand));
        int QN = s.nextInt();
        for (int i=0; i<QN; i++) {
            int num = s.nextInt();
            for (int j=0; j<num; j++) {
                n.doStuff();
            }
            int changeTo = s.nextInt();
            n.changeCommandStr(convertNumToCommandStr(changeTo));
        }
    }

    private static ICommand convertNumToCommand(int n) {
        switch (n) {
            case 1:
                return new LaptopCommand(new Laptop());
            case 2:
                return new PintuCommand(new Pintu());
            case 3:
                return new RumahCommand(new Rumah());
            default:
                return new LaptopCommand(new Laptop());
            }
    }

    private static String convertNumToCommandStr(int n) {
        switch (n) {
            case 1:
                return "laptop";
            case 2:
                return "pintu";
            case 3:
                return "rumah";
            default:
                return "laptop";
        }
    }

}
