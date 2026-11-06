const LIMIT: i64 = 1000000;

fn main() {
    let mut best_start: i64 = 1;
    let mut best_length: i64 = 1;
    let mut start: i64 = 1;
    while start < LIMIT {
        let mut n = start;
        let mut length: i64 = 1;
        while n != 1 {
            if n % 2 == 0 {
                n /= 2;
            } else {
                n = 3 * n + 1;
            }
            length += 1;
        }
        if length > best_length {
            best_length = length;
            best_start = start;
        }
        start += 1;
    }
    println!("{} {}", best_start, best_length);
}
