#include <stdint.h>
#include <stdio.h>

#define LIMIT 1000000

int main(void) {
	int64_t bestStart = 1;
	int64_t bestLength = 1;
	for (int64_t start = 1; start < LIMIT; start++) {
		int64_t n = start;
		int64_t length = 1;
		while (n != 1) {
			if (n % 2 == 0) {
				n = n / 2;
			} else {
				n = 3 * n + 1;
			}
			length++;
		}
		if (length > bestLength) {
			bestLength = length;
			bestStart = start;
		}
	}
	printf("%lld %lld\n", (long long)bestStart, (long long)bestLength);
	return 0;
}
