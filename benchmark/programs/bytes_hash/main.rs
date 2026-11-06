const SIZE: usize = 16777216;
const REPEAT: usize = 8;

fn main() {
    let mut buf = vec![0u8; SIZE];
    let mut x: u64 = 88172645463325252;
    for i in 0..SIZE {
        x ^= x << 13;
        x ^= x >> 7;
        x ^= x << 17;
        buf[i] = x as u8;
    }
    let mut h: u64 = 0xcbf29ce484222325;
    for _ in 0..REPEAT {
        for i in 0..SIZE {
            h ^= buf[i] as u64;
            h = h.wrapping_mul(1099511628211);
        }
    }
    println!("{}", h);
}
