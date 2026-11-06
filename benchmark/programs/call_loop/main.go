//go:build ignore

package main

import "fmt"

const N = 300000000

func add(a, b int64) int64 {
	return a + b
}

func main() {
	var total int64
	for i := int64(0); i < N; i++ {
		total = add(total, (i^total)&1023)
	}
	fmt.Println(total)
}
