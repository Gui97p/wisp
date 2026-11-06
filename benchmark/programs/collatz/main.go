//go:build ignore

package main

import "fmt"

const LIMIT = 1000000

func main() {
	var bestStart, bestLength int64 = 1, 1
	for start := int64(1); start < LIMIT; start++ {
		n := start
		var length int64 = 1
		for n != 1 {
			if n%2 == 0 {
				n = n / 2
			} else {
				n = 3*n + 1
			}
			length++
		}
		if length > bestLength {
			bestLength = length
			bestStart = start
		}
	}
	fmt.Println(bestStart, bestLength)
}
