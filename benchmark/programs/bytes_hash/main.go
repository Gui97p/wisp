//go:build ignore

package main

import "fmt"

const SIZE = 16777216
const REPEAT = 8

func main() {
	buf := make([]uint8, SIZE)
	var x uint64 = 88172645463325252
	for i := 0; i < SIZE; i++ {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		buf[i] = uint8(x)
	}
	var h uint64 = 0xcbf29ce484222325
	for r := 0; r < REPEAT; r++ {
		for i := 0; i < SIZE; i++ {
			h ^= uint64(buf[i])
			h *= 1099511628211
		}
	}
	fmt.Println(h)
}
