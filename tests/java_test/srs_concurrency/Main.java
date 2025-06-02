import java.security.SecureRandom;
import java.util.ArrayList;
import java.util.Scanner;
import java.util.Set;
import java.util.concurrent.CopyOnWriteArraySet;

public class Main {

    public static void main(String[] args) throws InterruptedException {
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

    public static void assertTC(boolean status) {
        if (status == true) {
            System.out.println('1');
        } else {
            System.out.println('0');
        }
    }

    private static Ans genRandArray(int n, int mins, int maks) {
        ArrayList<String> arr = new ArrayList<String>(n);
        SecureRandom rand = new SecureRandom();

        int cntOne = 0;
        int cntTwo = 0;
        int cntThree = 0;
        for (int i = 0; i < n; i++) {
            String strone = "";
            String strtwo = "";
            String strthree = "";
            String one = "onex";
            String two = "two";
            String three = "threes";
            int randone = rand.nextInt(200);
            int randtwo = rand.nextInt(150);
            int randthree = rand.nextInt(250);
            cntOne += randone;
            cntTwo += randtwo;
            cntThree += randthree;
            for (int j = 0; j < randone; j++) {
                strone += one;
            }

            for (int j = 0; j < randtwo; j++) {
                strtwo += two;
            }

            for (int j = 0; j < randthree; j++) {
                strthree += three;
            }

            arr.add(strone + strtwo + strthree);
        }

        Ans ans = new Ans();
        ans.arr = arr;
        ans.cntOne = cntOne;
        ans.cntTwo = cntTwo;
        ans.cntThree = cntThree;
        return ans;
    }

    private static void tc1() throws InterruptedException {
        int nWorkers = 1;
        Set<Thread> threadsUsed = new CopyOnWriteArraySet<>();
        Ans ans = genRandArray(1, 20, 50);
        ComplexWordCountServer arrayWord = new ComplexWordCountServer(nWorkers, ans.arr) {
            @Override
            protected int[] characterCountHelper(String str) {
                threadsUsed.add(Thread.currentThread());
                return super.characterCountHelper(str);
            }
        };
        arrayWord.countSpecialString();

        String ansStr = String.format(
            "one : %d, two : %d, three : %d\n",
            ans.cntOne,
            ans.cntTwo,
            ans.cntThree
        );
        assertTC(Math.min(nWorkers, ans.arr.size()) == threadsUsed.size());
        assertTC(ansStr.equals(arrayWord.toString()));
    }

    private static void tc2() throws InterruptedException {
        int nWorkers = 10;
        Set<Thread> threadsUsed = new CopyOnWriteArraySet<>();
        Ans ans = genRandArray(10, 20, 50);
        ComplexWordCountServer arrayWord = new ComplexWordCountServer(nWorkers, ans.arr) {
            @Override
            protected int[] characterCountHelper(String str) {
                threadsUsed.add(Thread.currentThread());
                return super.characterCountHelper(str);
            }
        };
        arrayWord.countSpecialString();

        String ansStr = String.format(
            "one : %d, two : %d, three : %d\n",
            ans.cntOne,
            ans.cntTwo,
            ans.cntThree
        );
        assertTC(Math.min(nWorkers, ans.arr.size()) == threadsUsed.size());
        assertTC(ansStr.equals(arrayWord.toString()));
    }

    private static void tc3() throws InterruptedException {
        int nWorkers = 20;
        Set<Thread> threadsUsed = new CopyOnWriteArraySet<>();
        Ans ans = genRandArray(20, 20, 50);
        ComplexWordCountServer arrayWord = new ComplexWordCountServer(nWorkers, ans.arr) {
            @Override
            protected int[] characterCountHelper(String str) {
                threadsUsed.add(Thread.currentThread());
                return super.characterCountHelper(str);
            }
        };
        arrayWord.countSpecialString();

        String ansStr = String.format(
            "one : %d, two : %d, three : %d\n",
            ans.cntOne,
            ans.cntTwo,
            ans.cntThree
        );
        assertTC(Math.min(nWorkers, ans.arr.size()) == threadsUsed.size());
        assertTC(ansStr.equals(arrayWord.toString()));
    }

    private static void tc4() throws InterruptedException {
        int nWorkers = 3;
        Set<Thread> threadsUsed = new CopyOnWriteArraySet<>();
        ArrayList<String> arr = new ArrayList<String>(3);
        String[] words = { "oner", "twotw", "hreeo" };
        for (int i = 0; i < words.length; i++) {
            arr.add(words[i]);
        }
        ComplexWordCountServer arrayWord = new ComplexWordCountServer(nWorkers, arr) {
            @Override
            protected int[] characterCountHelper(String str) {
                threadsUsed.add(Thread.currentThread());
                return super.characterCountHelper(str);
            }
        };
        arrayWord.countSpecialString();

        String ansStr = String.format("one : %d, two : %d, three : %d\n", 1, 2, 0);
        assertTC(Math.min(nWorkers, arr.size()) == threadsUsed.size());
        assertTC(ansStr.equals(arrayWord.toString()));
    }

    private static void tc5() throws InterruptedException {
        int nWorkers = 1;
        Set<Thread> threadsUsed = new CopyOnWriteArraySet<>();
        ArrayList<String> arr = new ArrayList<String>(1);
        String[] words = { "" };
        for (int i = 0; i < words.length; i++) {
            arr.add(words[i]);
        }
        ComplexWordCountServer arrayWord = new ComplexWordCountServer(nWorkers, arr) {
            @Override
            protected int[] characterCountHelper(String str) {
                threadsUsed.add(Thread.currentThread());
                return super.characterCountHelper(str);
            }
        };
        arrayWord.countSpecialString();

        String ansStr = String.format("one : %d, two : %d, three : %d\n", 0, 0, 0);
        assertTC(Math.min(nWorkers, arr.size()) == threadsUsed.size());
        assertTC(ansStr.equals(arrayWord.toString()));
    }
}
