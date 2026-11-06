const std = @import("std");

const SIZE: usize = 16777216;
const REPEAT: usize = 8;

pub fn main() void {
    const buf = std.heap.page_allocator.alloc(u8, SIZE) catch unreachable;
    defer std.heap.page_allocator.free(buf);
    var x: u64 = 88172645463325252;
    var i: usize = 0;
    while (i < SIZE) : (i += 1) {
        x ^= x << 13;
        x ^= x >> 7;
        x ^= x << 17;
        buf[i] = @truncate(x);
    }
    var h: u64 = 0xcbf29ce484222325;
    var r: usize = 0;
    while (r < REPEAT) : (r += 1) {
        i = 0;
        while (i < SIZE) : (i += 1) {
            h ^= @as(u64, buf[i]);
            h *%= 1099511628211;
        }
    }
    std.debug.print("{d}\n", .{h});
}
