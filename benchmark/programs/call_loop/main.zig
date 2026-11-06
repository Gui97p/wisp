const std = @import("std");

fn add(a: i64, b: i64) i64 {
    return a + b;
}

pub fn main() void {
    var total: i64 = 0;
    var i: i64 = 0;
    while (i < 300000000) : (i += 1) {
        total = add(total, (i ^ total) & 1023);
    }
    std.debug.print("{d}\n", .{total});
}
