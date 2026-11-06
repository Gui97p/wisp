#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

#define N 300

int main(void) {
	double *a = malloc(N * N * sizeof(double));
	double *b = malloc(N * N * sizeof(double));
	double *c = malloc(N * N * sizeof(double));
	for (int64_t i = 0; i < N; i++) {
		for (int64_t j = 0; j < N; j++) {
			a[i * N + j] = (double)((i + j) % 10);
			b[i * N + j] = (double)((i * j) % 10);
		}
	}
	for (int64_t i = 0; i < N; i++) {
		for (int64_t j = 0; j < N; j++) {
			double sum = 0.0;
			for (int64_t k = 0; k < N; k++) {
				sum += a[i * N + k] * b[k * N + j];
			}
			c[i * N + j] = sum;
		}
	}
	double total = 0.0;
	for (int64_t i = 0; i < N * N; i++) {
		total += c[i];
	}
	printf("%lld\n", (long long)total);
	free(a);
	free(b);
	free(c);
	return 0;
}
