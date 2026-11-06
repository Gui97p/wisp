const N: usize = 300;

fn main() {
    let mut a = vec![0.0f64; N * N];
    let mut b = vec![0.0f64; N * N];
    let mut c = vec![0.0f64; N * N];
    for i in 0..N {
        for j in 0..N {
            a[i * N + j] = ((i + j) % 10) as f64;
            b[i * N + j] = ((i * j) % 10) as f64;
        }
    }
    for i in 0..N {
        for j in 0..N {
            let mut sum = 0.0f64;
            for k in 0..N {
                sum += a[i * N + k] * b[k * N + j];
            }
            c[i * N + j] = sum;
        }
    }
    let mut total = 0.0f64;
    for i in 0..N * N {
        total += c[i];
    }
    println!("{}", total as i64);
}
