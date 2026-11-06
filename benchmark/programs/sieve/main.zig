const std = @import("std");

const N: usize = 50000000;

pub fn main() void {
    const composite = std.heap.page_allocator.alloc(u8, N + 1) catch unreachable;
    defer std.heap.page_allocator.free(composite);
    @memset(composite, 0);
    var count: i64 = 0;
    var i: usize = 2;
    while (i <= N) : (i += 1) {
        if (composite[i] == 0) {
            count += 1;
            var j: usize = i * i;
            while (j <= N) : (j += i) {
                composite[j] = 1;
            }
        }
    }
    std.debug.print("{d}\n", .{count});
}
