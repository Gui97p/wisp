const std = @import("std");

const LIMIT: i64 = 1000000;

pub fn main() void {
    var best_start: i64 = 1;
    var best_length: i64 = 1;
    var start: i64 = 1;
    while (start < LIMIT) : (start += 1) {
        var n: i64 = start;
        var length: i64 = 1;
        while (n != 1) {
            if (@rem(n, 2) == 0) {
                n = @divTrunc(n, 2);
            } else {
                n = 3 * n + 1;
            }
            length += 1;
        }
        if (length > best_length) {
            best_length = length;
            best_start = start;
        }
    }
    std.debug.print("{d} {d}\n", .{ best_start, best_length });
}
