/* The C side of the interop example. Build it into an object file first:
 *
 *     cc -c -O1 -fno-stack-protector -fno-pic -ffreestanding lib.c -o lib.o
 *
 * wisp.toml lists lib.o under `link`, so `wisp run` hands it to the linker.
 */

int add(int a, int b) {
    return a + b;
}

int twice(int x) {
    return 2 * x;
}

int sum_bytes(const char *bytes, int count) {
    int total = 0;
    for (int i = 0; i < count; i++) {
        total += (unsigned char)bytes[i];
    }
    return total;
}

/* Declared here, defined in Wisp (main.wsp): C calls back into the program. */
extern int wisp_scale(int value, int factor);

int scale_by_three(int value) {
    return wisp_scale(value, 3);
}
