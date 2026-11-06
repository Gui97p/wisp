//go:build ignore

package main

import "fmt"

const N = 300

func main() {
	a := make([]float64, N*N)
	b := make([]float64, N*N)
	c := make([]float64, N*N)
	for i := 0; i < N; i++ {
		for j := 0; j < N; j++ {
			a[i*N+j] = float64((i + j) % 10)
			b[i*N+j] = float64((i * j) % 10)
		}
	}
	for i := 0; i < N; i++ {
		for j := 0; j < N; j++ {
			sum := 0.0
			for k := 0; k < N; k++ {
				sum += a[i*N+k] * b[k*N+j]
			}
			c[i*N+j] = sum
		}
	}
	total := 0.0
	for i := 0; i < N*N; i++ {
		total += c[i]
	}
	fmt.Println(int64(total))
}
