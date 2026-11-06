const N: i64 = 300000000;

fn add(a: i64, b: i64) -> i64 {
    a + b
}

fn main() {
    let mut total: i64 = 0;
    let mut i: i64 = 0;
    while i < N {
        total = add(total, (i ^ total) & 1023);
        i += 1;
    }
    println!("{}", total);
}
