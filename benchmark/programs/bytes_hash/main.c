#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

#define SIZE 16777216
#define REPEAT 8

int main(void) {
	uint8_t *buf = malloc(SIZE);
	uint64_t x = 88172645463325252ULL;
	for (int64_t i = 0; i < SIZE; i++) {
		x ^= x << 13;
		x ^= x >> 7;
		x ^= x << 17;
		buf[i] = (uint8_t)x;
	}
	uint64_t h = 0xcbf29ce484222325ULL;
	for (int r = 0; r < REPEAT; r++) {
		for (int64_t i = 0; i < SIZE; i++) {
			h ^= (uint64_t)buf[i];
			h *= 1099511628211ULL;
		}
	}
	printf("%llu\n", (unsigned long long)h);
	free(buf);
	return 0;
}
