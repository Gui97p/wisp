const std = @import("std");

const N: usize = 300;

pub fn main() void {
    const alloc = std.heap.page_allocator;
    const a = alloc.alloc(f64, N * N) catch unreachable;
    const b = alloc.alloc(f64, N * N) catch unreachable;
    const c = alloc.alloc(f64, N * N) catch unreachable;
    defer alloc.free(a);
    defer alloc.free(b);
    defer alloc.free(c);
    var i: usize = 0;
    while (i < N) : (i += 1) {
        var j: usize = 0;
        while (j < N) : (j += 1) {
            a[i * N + j] = @floatFromInt((i + j) % 10);
            b[i * N + j] = @floatFromInt((i * j) % 10);
        }
    }
    i = 0;
    while (i < N) : (i += 1) {
        var j: usize = 0;
        while (j < N) : (j += 1) {
            var sum: f64 = 0.0;
            var k: usize = 0;
            while (k < N) : (k += 1) {
                sum += a[i * N + k] * b[k * N + j];
            }
            c[i * N + j] = sum;
        }
    }
    var total: f64 = 0.0;
    i = 0;
    while (i < N * N) : (i += 1) {
        total += c[i];
    }
    std.debug.print("{d}\n", .{@as(i64, @intFromFloat(total))});
}
