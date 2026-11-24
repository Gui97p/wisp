/* The C side of the interop example. Build it into an object file first:
 *
 *     cc -c -O1 -fno-stack-protector -fno-pic -ffreestanding lib.c -o lib.o
 *
 * wisp.toml lists lib.o under `link`, so `wisp run` hands it to the linker.
 */

int lib_add(int a, int b) {
    return a + b;
}

int lib_twice(int x) {
    return 2 * x;
}

int lib_sum_bytes(const char *bytes, int count) {
    int total = 0;
    for (int i = 0; i < count; i++) {
        total += (unsigned char)bytes[i];
    }
    return total;
}

/* Declared here, defined in Wisp (main.wsp): C calls back into the program. */
extern int wisp_scale(int value, int factor);

int lib_scale_by_three(int value) {
    return wisp_scale(value, 3);
}

int lib_checked(int value) {
    return value < 0 ? -1 : value;
}

const char *lib_find(int key) {
    return key == 7 ? "seven" : (const char *)0;
}

typedef struct counter {
    int value;
} counter;

static counter the_counter;

counter *lib_counter_get(void) {
    return &the_counter;
}

void lib_counter_add(counter *c, int n) {
    c->value += n;
}

int lib_counter_value(const counter *c) {
    return c->value;
}

int lib_sum_ints(const int *values, unsigned long count) {
    int total = 0;
    for (unsigned long i = 0; i < count; i++) {
        total += values[i];
    }
    return total;
}
