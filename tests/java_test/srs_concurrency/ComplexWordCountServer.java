import java.util.ArrayList;

public class ComplexWordCountServer {

    private int nWorkers;
    private ArrayList<String> array;
    int[] res = new int[26];

    ComplexWordCountServer(int nWorkers, ArrayList<String> array) {
        this.nWorkers = nWorkers;
        this.array = array;
    }

    public void countSpecialString() throws InterruptedException {
        ArrayList<Thread> threads = new ArrayList<>();

        for (String str : array) {
            Thread t = new Thread(() -> {
                int[] localCount = characterCountHelper(str);
                synchronized (res) {
                    for (int i = 0; i < 26; i++) {
                        res[i] += localCount[i];
                    }
                }
            });
            threads.add(t);
            t.start();
        }

        for (Thread t : threads) {
            t.join();
        }
    }

    protected int[] characterCountHelper(String str) {
        int[] count = new int[26];
        for (char c : str.toCharArray()) {
            count[c - 'a']++;
        }
        return count;
    }

    public String toString() {
        int one = Math.min(res['o' - 'a'], Math.min(res['n' - 'a'], res['e' - 'a']));
        int two = Math.min(res['t' - 'a'], Math.min(res['w' - 'a'], res['o' - 'a']));
        int three = Math.min(Math.min(res['t' - 'a'], res['h' - 'a']),
                Math.min(res['r' - 'a'], Math.min(res['e' - 'a'] / 2, res['e' - 'a'] / 2))); // 'e' counted twice

        return String.format("one : %d, two : %d, three : %d\n", one, two, three);
    }
} 
