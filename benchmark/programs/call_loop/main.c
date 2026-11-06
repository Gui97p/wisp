#include <stdint.h>
#include <stdio.h>

#define N 300000000

static int64_t add(int64_t a, int64_t b) {
	return a + b;
}

int main(void) {
	int64_t total = 0;
	for (int64_t i = 0; i < N; i++) {
		total = add(total, (i ^ total) & 1023);
	}
	printf("%lld\n", (long long)total);
	return 0;
}
