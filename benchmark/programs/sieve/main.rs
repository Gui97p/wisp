const N: usize = 50000000;

fn main() {
    let mut composite = vec![0u8; N + 1];
    let mut count: i64 = 0;
    let mut i: usize = 2;
    while i <= N {
        if composite[i] == 0 {
            count += 1;
            let mut j = i * i;
            while j <= N {
                composite[j] = 1;
                j += i;
            }
        }
        i += 1;
    }
    println!("{}", count);
}
