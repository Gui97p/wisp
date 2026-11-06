//go:build ignore

package main

import "fmt"

const N = 50000000

func main() {
	composite := make([]uint8, N+1)
	var count int64
	for i := int64(2); i <= N; i++ {
		if composite[i] == 0 {
			count++
			for j := i * i; j <= N; j += i {
				composite[j] = 1
			}
		}
	}
	fmt.Println(count)
}
