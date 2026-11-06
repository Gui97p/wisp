#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

#define N 50000000

int main(void) {
	uint8_t *composite = calloc(N + 1, 1);
	int64_t count = 0;
	for (int64_t i = 2; i <= N; i++) {
		if (composite[i] == 0) {
			count++;
			for (int64_t j = i * i; j <= N; j += i) {
				composite[j] = 1;
			}
		}
	}
	printf("%lld\n", (long long)count);
	free(composite);
	return 0;
}
