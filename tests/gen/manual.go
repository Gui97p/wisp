package main

type manualCase struct {
	dir      string
	src      string
	expected []string
}

var manual = []manualCase{
	{dir: "big_literal", src: `func id(int64 x) int64 {
    return x;
}

func eighth(int64 a, int64 b, int64 c, int64 d, int64 e, int64 f, int64 g, int64 h) int64 {
    return h;
}

func big() int64 {
    return 5000000000;
}

func main() {
    int64 a = 5000000000;
    check(a == 5000000000, "biglit init and compare right");
    check(5000000000 == a, "biglit compare left");
    check(a != 4999999999, "biglit not equal");
    int64 b = a + 5000000000;
    check(b == 10000000000, "biglit add right operand");
    int64 c = 5000000000 + a;
    check(c == 10000000000, "biglit add left operand");
    int64 d = a - 4999999999;
    check(d == 1, "biglit sub right operand");
    int64 e = 20000000000 - a;
    check(e == 15000000000, "biglit sub left operand");
    int64 f = a * 3;
    check(f == 15000000000, "biglit mul small literal");
    a += 5000000000;
    check(a == 10000000000, "biglit compound add");
    a -= 1;
    check(a == 9999999999, "biglit compound sub small");
    a = 4294967296;
    check(a == 4294967296, "biglit two to the 32");
    a = 2147483648;
    check(a == 2147483648, "biglit two to the 31");
    a = 2147483647;
    a += 1;
    check(a == 2147483648, "biglit crossing 32 bit boundary by add");
    check(id(5000000000) == 5000000000, "biglit as argument");
    check(big() == 5000000000, "biglit as return value");
    check(eighth(1, 2, 3, 4, 5, 6, 7, 5000000000) == 5000000000, "biglit as stack argument");
    uint64 u = 18446744073709551615;
    check(u == 18446744073709551615, "biglit uint64 max equality");
    uint64 h = 9223372036854775808;
    check(h == 9223372036854775808, "biglit uint64 two to the 63");
    check(u != h, "biglit uint64 distinct");
    uint64 w = h + h;
    check(w == 0, "biglit uint64 wraps to zero");
    uint64 x = 0;
    x -= 1;
    check(x == 18446744073709551615, "biglit uint64 zero minus one");
}
`},

	{dir: "abi_narrow", src: `func id8(int8 x) int8 {
    return x;
}

func idu8(uint8 x) uint8 {
    return x;
}

func id16(int16 x) int16 {
    return x;
}

func idu16(uint16 x) uint16 {
    return x;
}

func id32(int32 x) int32 {
    return x;
}

func idu32(uint32 x) uint32 {
    return x;
}

func idb(bool x) bool {
    return x;
}

func add8(int8 a, int8 b) int8 {
    return a + b;
}

func gt8(int8 a, int8 b) bool {
    return a > b;
}

func ugt8(uint8 a, uint8 b) bool {
    return a > b;
}

func last8(int8 a, int8 b, int8 c, int8 d, int8 e, int8 f, int8 g, int8 h) int8 {
    return g + h;
}

func mix(int8 a, uint16 b, int32 c, uint8 d, int64 e) int64 {
    int8 m3 = 0;
    m3 -= 3;
    check(a == m3, "abi mix int8 arg");
    check(b == 60000, "abi mix uint16 arg");
    check(c == 70000, "abi mix int32 arg");
    check(d == 200, "abi mix uint8 arg");
    check(e == 5000000000, "abi mix int64 arg");
    return e;
}

func two8() (int8, uint8) {
    int8 m = 0;
    m -= 5;
    return m, 250;
}

func three() (int8, int16, int32) {
    int8 a = 0;
    a -= 7;
    int16 b = 0;
    b -= 300;
    return a, b, 100000;
}

func main() {
    int8 m1 = 0;
    m1 -= 1;
    check(id8(m1) == m1, "abi id int8 negative");
    check(idu8(255) == 255, "abi id uint8 max");
    int16 m2 = 0;
    m2 -= 30000;
    check(id16(m2) == m2, "abi id int16 negative");
    check(idu16(65535) == 65535, "abi id uint16 max");
    int32 m3 = 0;
    m3 -= 2000000000;
    check(id32(m3) == m3, "abi id int32 negative");
    check(idu32(4294967295) == 4294967295, "abi id uint32 max");
    check(idb(true), "abi id bool true");
    int8 m56 = 0;
    m56 -= 56;
    check(add8(100, 100) == m56, "abi add8 wraps");
    check(gt8(1, m1), "abi gt8 signed");
    check(ugt8(255, 1), "abi ugt8 unsigned high");
    check(ugt8(200, 100), "abi ugt8 200 over 100");
    int8 m3b = 0;
    m3b -= 3;
    check(mix(m3b, 60000, 70000, 200, 5000000000) == 5000000000, "abi mix returns int64");
    check(last8(1, 2, 3, 4, 5, 6, 7, 8) == 15, "abi narrow stack args");
    int8 m5 = 0;
    m5 -= 5;
    let p, q = two8();
    check(p == m5, "abi two8 first int8");
    check(q == 250, "abi two8 second uint8");
    int8 m7 = 0;
    m7 -= 7;
    int16 m300 = 0;
    m300 -= 300;
    let x8, x16, x32 = three();
    check(x8 == m7, "abi three sret int8");
    check(x16 == m300, "abi three sret int16");
    check(x32 == 100000, "abi three sret int32");
}
`, expected: []string{
		"abi id int8 negative", "abi id uint8 max", "abi id int16 negative", "abi id uint16 max",
		"abi id int32 negative", "abi id uint32 max", "abi id bool true", "abi add8 wraps",
		"abi gt8 signed", "abi ugt8 unsigned high", "abi ugt8 200 over 100",
		"abi mix int8 arg", "abi mix uint16 arg", "abi mix int32 arg", "abi mix uint8 arg", "abi mix int64 arg",
		"abi mix returns int64", "abi narrow stack args", "abi two8 first int8", "abi two8 second uint8",
		"abi three sret int8", "abi three sret int16", "abi three sret int32",
	}},

	{dir: "logic_bool", src: `func yes() bool {
    emit("yes called");
    return true;
}

func no() bool {
    emit("no called");
    return false;
}

func main() {
    bool t = true;
    bool f = false;
    check(t && t, "logic and tt");
    checkEq(t && f, false, "logic and tf");
    checkEq(f && t, false, "logic and ft");
    checkEq(f && f, false, "logic and ff");
    check(t || t, "logic or tt");
    check(t || f, "logic or tf");
    check(f || t, "logic or ft");
    checkEq(f || f, false, "logic or ff");
    checkEq(no() && yes(), false, "logic and short circuits");
    check(yes() || no(), "logic or short circuits");
    check(yes() && yes(), "logic and evaluates both");
    checkEq(no() || no(), false, "logic or evaluates both");
    int a = 5;
    check(a > 3 && a < 10, "logic range check");
    check(a < 3 || a > 4, "logic or of comparisons");
    bool both = a > 3 && a < 10;
    check(both, "logic stored in bool variable");
}
`, expected: []string{
		"logic and tt", "logic and tf", "logic and ft", "logic and ff",
		"logic or tt", "logic or tf", "logic or ft", "logic or ff",
		"no called", "logic and short circuits",
		"yes called", "logic or short circuits",
		"yes called", "yes called", "logic and evaluates both",
		"no called", "no called", "logic or evaluates both",
		"logic range check", "logic or of comparisons", "logic stored in bool variable",
	}},

	{dir: "precedence", src: `func main() {
    int a = 2;
    int b = 3;
    int c = 4;
    check(a + b * c == 14, "prec mul before add");
    check((a + b) * c == 20, "prec parens override");
    check(c - b - a + 1 == 0, "prec sub left associative");
    check((a + 3) * 4 - (a - 1) == 19, "prec nested parens");
    check(a + b < c + 2 && b * 2 == 6, "prec compare then logic");
    check(a * b + c * a == 14, "prec two products");
    check(((a + b) * (c + a)) - ((a * b) + (c - b)) == 23, "prec deep tree");
    check(100 - 10 - 5 == 85, "prec literal chain sub");
    check(2 * 3 + 4 * 5 == 26, "prec literal chain mul add");
}
`},

	{dir: "register_pressure", src: `func idi(int x) int {
    return x;
}

func main() {
    int a1 = 1;
    int a2 = 2;
    int a3 = 3;
    int a4 = 4;
    int a5 = 5;
    int a6 = 6;
    int a7 = 7;
    int a8 = 8;
    int a9 = 9;
    int a10 = 10;
    int a11 = 11;
    int a12 = 12;
    int s = a1 + (a2 + (a3 + (a4 + (a5 + (a6 + (a7 + (a8 + (a9 + (a10 + (a11 + a12))))))))));
    check(s == 78, "pressure right nested sum of 12");
    int p = (a1 + a2) * (a3 + a4) + (a5 + a6) * (a7 + a8) + (a9 + a10) * (a11 + a12);
    check(p == 623, "pressure sum of products");
    check(a1 + (a2 + (a3 + (idi(a4) + a5))) == 15, "pressure with call in the middle");
    check(idi(a1) + idi(a2) + idi(a3) + idi(a4) + idi(a5) + idi(a6) == 21, "pressure chain of calls");
}
`},

	{dir: "untyped_const", src: `func main() {
    int x = 2 * 3 + 4;
    check(x == 10, "const arithmetic into int");
    int8 c = 100 + 20;
    check(c == 120, "const arithmetic into int8");
    uint8 u = 200 + 55;
    check(u == 255, "const arithmetic into uint8");
    int64 big = 1000000 * 1000000;
    check(big == 1000000000000, "const arithmetic into int64");
    int w = 10 - 20;
    check(w + 10 == 0, "const negative result");
}
`},

	{dir: "literal_left", src: `func main() {
    int8 i = 5;
    uint8 u = 200;
    int32 w = 7;
    uint64 big = 18446744073709551615;
    check(0 < i, "literal left lt int8");
    checkEq(10 < i, false, "literal left lt int8 false");
    check(5 == i, "literal left eq int8");
    check(250 > u, "literal left gt uint8");
    check(1 < u, "literal left lt uint8");
    check(255 >= u, "literal left ge uint8");
    check(100 + i == 105, "literal left add int8");
    check(100 - i == 95, "literal left sub int8");
    check(3 * i == 15, "literal left mul int8");
    check(1 < big, "literal left lt uint64 high");
    check(18446744073709551615 == big, "literal left eq uint64 max");
}
`},

	{dir: "alias_types", src: `func main() {
    int a = 9223372036854775807;
    a += 1;
    int m = 0;
    m -= 9223372036854775807;
    m -= 1;
    check(a == m, "alias int wraps like int64");
    uint u = 0;
    u -= 1;
    check(u == 18446744073709551615, "alias uint wraps like uint64");
    char c = 65;
    c += 1;
    check(c == 66, "alias char arithmetic");
    char d = 255;
    d += 1;
    check(d == 0, "alias char wraps like uint8");
}
`},

	{dir: "cast_literal", src: `func main() {
    uint8 a = 300 as uint8;
    check(a == 44, "castlit 300 as uint8");
    int8 b = 200 as int8;
    check(b + 56 == 0, "castlit 200 as int8");
    uint16 c = 70000 as uint16;
    check(c == 4464, "castlit 70000 as uint16");
    int32 d = 4294967295 as int32;
    check(d + 1 == 0, "castlit 4294967295 as int32");
    uint32 e = 5000000000 as uint32;
    check(e == 705032704, "castlit 5000000000 as uint32");
    int64 f = 18446744073709551615 as int64;
    check(f + 1 == 0, "castlit uint64 max as int64");
}
`},

	{dir: "float_future", src: `func main() {
    float64 f = 3.9;
    int i = f as int;
    check(i == 3, "float to int truncates");
    int j = 5;
    float64 g = j as float64;
    check(g == 5.0, "int to float");
    float64 h = 2.5;
    check(h * 2.0 == 5.0, "float mul");
    check(h + 0.5 == 3.0, "float add");
    check(h > 2.0, "float compare");
    float64 neg = 0.0 - 1.5;
    int k = neg as int;
    check(k + 1 == 0, "float negative to int truncates toward zero");
}
`},

	{dir: "big_literal_bitwise", src: `func main() {
    int64 hi = 0x7FFFFFFF00000000;
    int64 lo = 0xFFFFFFFF;
    check((hi | lo) == 0x7FFFFFFFFFFFFFFF, "biglit bitwise or");
    check((hi & lo) == 0, "biglit bitwise and");
    check((hi ^ lo) == 0x7FFFFFFFFFFFFFFF, "biglit bitwise xor");
}
`},

	{dir: "precedence_divmod", src: `func main() {
    int a = 2;
    int b = 3;
    int c = 4;
    check(c / a * b == 6, "prec div mul left associative");
    check(c % b + a == 3, "prec mod before add");
    check(100 / 10 / 5 == 2, "prec literal chain div");
}
`},

	{dir: "literal_left_divmod", src: `func main() {
    int8 i = 5;
    int32 w = 7;
    check(100 / i == 20, "literal left div int8");
    check(7 % w == 0, "literal left mod int32");
    check(20 / w == 2, "literal left div int32");
}
`},

	{dir: "untyped_const_divmod", src: `func main() {
    int y = 7 / 2;
    check(y == 3, "const integer division truncates");
    int z = 7 % 4;
    check(z == 3, "const modulo");
}
`},

	{dir: "logic_not", src: `func main() {
    bool t = true;
    bool f = false;
    int a = 5;
    check(!f, "logic not false");
    checkEq(!t, false, "logic not true");
    check(!(a == 6), "logic not of comparison");
    check(!!t, "logic double not");
}
`},

	{dir: "register_spill", src: `func main() {
    int a1 = 1;
    int a2 = 2;
    int a3 = 3;
    int a4 = 4;
    int a5 = 5;
    int a6 = 6;
    int a7 = 7;
    int a8 = 8;
    int a9 = 9;
    int a10 = 10;
    int a11 = 11;
    int a12 = 12;
    int q = (a1 + a2) + ((a3 + a4) + ((a5 + a6) + ((a7 + a8) + ((a9 + a10) + ((a11 + a12) + ((a1 + a2) + ((a3 + a4) + ((a5 + a6) + ((a7 + a8) + ((a9 + a10) + (a11 + a12)))))))))));
    check(q == 156, "spill twelve pending registers");
}
`},

	{dir: "shift_count_overflow", src: `func main() {
    uint8 one8 = 1;
    uint8 n8 = 8;
    check((one8 << n8) == 0, "shiftovf uint8 shl by 8 is zero");
    uint8 n9 = 9;
    check((one8 << n9) == 0, "shiftovf uint8 shl by 9 is zero");
    uint8 top8 = 128;
    check((top8 >> n8) == 0, "shiftovf uint8 shr by 8 is zero");
    int8 neg8 = 0;
    neg8 -= 1;
    int8 m8 = 8;
    check((neg8 >> m8) == neg8, "shiftovf int8 shr by 8 keeps sign");
    uint32 one32 = 1;
    uint32 n32 = 32;
    check((one32 << n32) == 0, "shiftovf uint32 shl by 32 is zero");
    check((one32 >> n32) == 0, "shiftovf uint32 shr by 32 is zero");
    uint64 one64 = 1;
    uint64 n64 = 64;
    check((one64 << n64) == 0, "shiftovf uint64 shl by 64 is zero");
    uint64 n70 = 70;
    check((one64 << n70) == 0, "shiftovf uint64 shl by 70 is zero");
    int64 neg64 = 0;
    neg64 -= 1;
    int64 m64 = 64;
    check((neg64 >> m64) == neg64, "shiftovf int64 shr by 64 keeps sign");
}
`},

	{dir: "flow_for_range", src: `func main() {
    int sum = 0;
    for i in 1..10 {
        sum += i;
    }
    check(sum == 55, "for inclusive range");
    int stepped = 0;
    for i in 1..10:4 {
        stepped += i;
    }
    check(stepped == 15, "for range with step");
    int hi = 5;
    int bounded = 0;
    for i in 1..hi {
        bounded += i;
    }
    check(bounded == 15, "for range with variable bound");
    int count = 0;
    for 3 {
        count++;
    }
    check(count == 3, "for repeat count");
    int big = 0;
    for i in 1..100000 {
        big += i;
    }
    check(big == 5000050000, "for large range sum past 32 bits");
}
`},

	{dir: "flow_loop", src: `func main() {
    int left = 3;
    int passes = 0;
    loop left != 0 {
        left--;
        passes++;
    }
    check(passes == 3, "loop while condition");
    int tries = 0;
    loop {
        tries++;
    } until tries == 3;
    check(tries == 3, "loop until runs body first");
    int once = 0;
    loop {
        once++;
    } until true;
    check(once == 1, "loop until true runs exactly once");
    int n = 0;
    int kept = 0;
    loop {
        n++;
        if n == 2 {
            continue;
        }
        if n > 4 {
            break;
        }
        kept += n;
    }
    check(kept == 8, "loop break and continue");
}
`},

	{dir: "flow_labels", src: `func main() {
    int inner = 0;
    :outer loop {
        loop {
            inner++;
            break outer;
        }
    }
    check(inner == 1, "label break outer");
    int total = 0;
    :rows for i in 1..3 {
        for j in 1..3 {
            if j == 2 {
                continue rows;
            }
            total += i * 10 + j;
        }
    }
    check(total == 63, "label continue outer");
    int nested = 0;
    for i in 1..4 {
        for j in 1..4 {
            nested += i * j;
        }
    }
    check(nested == 100, "nested for loops");
}
`},

	{dir: "flow_switch_stmt", src: `func classify(int v) int {
    int r = 0;
    switch v {
        case 1:
            r = 10;
        case 2..5:
            r = 20;
        default:
            r = 30;
    }
    return r;
}

func main() {
    check(classify(1) == 10, "switch exact case");
    check(classify(3) == 20, "switch range case");
    check(classify(5) == 20, "switch range upper bound");
    check(classify(6) == 30, "switch default");
    check(classify(0) == 30, "switch below range");
    int8 small = 4;
    int r = 0;
    switch small {
        case 4:
            r = 1;
        default:
            r = 2;
    }
    check(r == 1, "switch on int8");
}
`},

	{dir: "flow_switch_expr", src: `func pick(int v) int {
    return switch v {
        case 1 => 100,
        case 2 => 200,
        default => 0,
    };
}

func main() {
    check(pick(1) == 100, "switch expr first case");
    check(pick(2) == 200, "switch expr second case");
    check(pick(9) == 0, "switch expr default");
    int v = 2;
    let doubled = switch v {
        case 2 => 4,
        default => 0,
    };
    check(doubled == 4, "switch expr let");
    int assigned = 0;
    assigned = switch v {
        case 2 => 7,
        default => 0,
    };
    check(assigned == 7, "switch expr assignment");
}
`},

	{dir: "flow_if_chain", src: `func grade(int s) int {
    if s >= 90 {
        return 4;
    } else if s >= 80 {
        return 3;
    } else if s >= 70 {
        return 2;
    } else {
        return 1;
    }
}

func main() {
    check(grade(95) == 4, "if chain first");
    check(grade(85) == 3, "if chain second");
    check(grade(75) == 2, "if chain third");
    check(grade(10) == 1, "if chain else");
    int x = 0;
    if true {
        x = 1;
    }
    check(x == 1, "if constant true");
    if false {
        x = 2;
    }
    check(x == 1, "if constant false");
}
`},
}
