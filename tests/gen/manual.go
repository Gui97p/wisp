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
    int big = 300;
    uint8 a = uint8(big);
    check(a == 44, "castlit 300 to uint8");
    int two = 200;
    int8 b = int8(two);
    check(b + 56 == 0, "castlit 200 to int8");
    int wide = 70000;
    uint16 c = uint16(wide);
    check(c == 4464, "castlit 70000 to uint16");
    int64 huge = 4294967295;
    int32 d = int32(huge);
    check(d + 1 == 0, "castlit 4294967295 to int32");
    int64 five = 5000000000;
    uint32 e = uint32(five);
    check(e == 705032704, "castlit 5000000000 to uint32");
    uint64 top = 18446744073709551615;
    int64 f = int64(top);
    check(f + 1 == 0, "castlit uint64 max to int64");
    uint8 g = uint8(200);
    check(g == 200, "castlit constant that fits");
    int8 h = int8(0 + 100);
    check(h == 100, "castlit constant expression");
}
`},

	{dir: "float_future", src: `func main() {
    float64 f = 3.9;
    int i = int(f);
    check(i == 3, "float to int truncates");
    int j = 5;
    float64 g = float64(j);
    check(g == 5.0, "int to float");
    float64 h = 2.5;
    check(h * 2.0 == 5.0, "float mul");
    check(h + 0.5 == 3.0, "float add");
    check(h > 2.0, "float compare");
    float64 neg = 0.0 - 1.5;
    int k = int(neg);
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
    uint8 n33 = 33;
    check((one8 << n33) == 0, "shiftovf uint8 shl by 33 is zero");
    uint8 n255 = 255;
    check((top8 >> n255) == 0, "shiftovf uint8 shr by 255 is zero");
    int8 pos8 = 100;
    int8 m33 = 33;
    check((pos8 >> m33) == 0, "shiftovf int8 shr positive by 33 is zero");
    check((neg8 >> m33) == neg8, "shiftovf int8 shr negative by 33 keeps sign");
    uint16 one16 = 1;
    uint16 n16 = 16;
    check((one16 << n16) == 0, "shiftovf uint16 shl by 16 is zero");
    uint16 n40 = 40;
    check((one16 << n40) == 0, "shiftovf uint16 shl by 40 is zero");
    uint32 n33b = 33;
    check((one32 << n33b) == 0, "shiftovf uint32 shl by 33 is zero");
    int32 pos32 = 100;
    int32 m32 = 32;
    check((pos32 >> m32) == 0, "shiftovf int32 shr positive by 32 is zero");
    uint64 top64 = 9223372036854775808;
    check((top64 >> n64) == 0, "shiftovf uint64 shr by 64 is zero");
    int64 pos64 = 5;
    check((pos64 >> m64) == 0, "shiftovf int64 shr positive by 64 is zero");
    check((one32 << 32) == 0, "shiftovf uint32 shl literal 32 is zero");
    check((one64 << 64) == 0, "shiftovf uint64 shl literal 64 is zero");
    check((pos32 >> 40) == 0, "shiftovf int32 shr literal 40 is zero");
    check((neg64 >> 70) == neg64, "shiftovf int64 shr literal 70 keeps sign");
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

	{dir: "shift_registers", src: `func main() {
    uint32 a1 = 1;
    uint32 a2 = 2;
    uint32 b = 5;
    uint32 k = 1;
    check((a1 + a2) + (b << k) == 13, "shiftreg left operand lands in cx");
    int32 c1 = 1;
    int32 c2 = 2;
    int32 d = 20;
    int32 m = 2;
    check((c1 + c2) + (d >> m) == 8, "shiftreg signed shr with ax busy");
    uint32 e1 = 3;
    uint32 e2 = 4;
    uint32 f = 5;
    check((a1 + a2) + ((e1 + e2) + (f << k)) == 20, "shiftreg cx live across shift");
    uint32 n40 = 40;
    check((a1 + a2) + ((e1 + e2) + (f << n40)) == 10, "shiftreg zero path with cx live");
    check((a1 + a2) + (b << 40) == 3, "shiftreg literal overflow with ax busy");
    check((c1 + c2) + (d >> 40) == 3, "shiftreg signed literal overflow with ax busy");
    uint16 one = 1;
    uint16 n256 = 256;
    check((one << n256) == 0, "shiftreg uint16 count 256");
    uint16 n512 = 512;
    check((one << n512) == 0, "shiftreg uint16 count 512");
    uint64 one64 = 1;
    uint64 big = 256;
    check((one64 << big) == 0, "shiftreg uint64 count 256");
    uint64 k64 = 3;
    check((one64 << k64) == 8, "shiftreg uint64 count 3");
    check(((a1 + a2) << k) == 6, "shiftreg shifted value is a register");
    check((b << k) + (b << k) == 20, "shiftreg two shifts in one expression");
}
`},

	{dir: "conv_implicit", src: `enum Color {
    Red,
    Green,
    Blue
}

func take64(int64 x) int64 {
    return x;
}

func ret64(int8 x) int64 {
    return x;
}

func main() {
    int8 a = 100;
    int16 b = a;
    check(b == 100, "implicit assign int8 to int16");
    int64 c = b;
    check(c == 100, "implicit assign int16 to int64");
    uint8 u = 200;
    int16 s = u;
    check(s == 200, "implicit assign uint8 to int16");
    uint16 w = u;
    check(w == 200, "implicit assign uint8 to uint16");
    uint32 big32 = 4000000000;
    int64 asInt64 = big32;
    check(asInt64 == 4000000000, "implicit assign uint32 to int64");
    int32 m = 7;
    int64 n = 1000;
    check(m + n == 1007, "implicit binary int32 and int64");
    int64 sum = m + n;
    check(sum == 1007, "implicit binary result type");
    int8 x8 = 10;
    int16 y16 = 300;
    int16 prod = x8 * y16;
    check(prod == 3000, "implicit binary mul int8 and int16");
    check(x8 < y16, "implicit compare int8 and int16");
    check(y16 > x8, "implicit compare int16 and int8");
    uint8 u8 = 250;
    int8 i8 = 5;
    int16 mixed = u8 + i8;
    check(mixed == 255, "implicit mixed signedness widens to int16");
    uint32 u32 = 4000000000;
    int32 i32 = 1;
    int64 mixed64 = u32 + i32;
    check(mixed64 == 4000000001, "implicit mixed signedness widens to int64");
    check(take64(a) == 100, "implicit argument int8 to int64");
    check(ret64(a) == 100, "implicit return int8 to int64");
    int64 acc = 1;
    int8 inc = 5;
    acc += inc;
    check(acc == 6, "implicit compound assign value widens");
    Color col = Green;
    int code = col;
    check(code == 1, "implicit enum to int");
    int64 wide = col;
    check(wide == 1, "implicit enum to int64");
    check(col == code, "implicit enum compared with int");
    check(code == col, "implicit int compared with enum");
    char ch = 'a';
    int chi = ch;
    check(chi == 97, "implicit char to int");
}
`},

	{dir: "conv_explicit", src: `type UserID int;

enum Level {
    Low,
    Mid,
    High
}

func main() {
    int64 big = 300;
    uint8 n = uint8(big);
    check(n == 44, "conv narrowing wraps");
    uint8 u = 200;
    int8 s = int8(u);
    check(s + 56 == 0, "conv sign change wraps");
    int64 wide = int64(s);
    check(wide + 56 == 0, "conv widening keeps sign");
    uint16 z = uint16(s);
    check(z == 65480, "conv signed to wider unsigned");
    UserID uid = 42;
    int raw = int(uid);
    check(raw == 42, "conv named to underlying");
    UserID back = UserID(raw);
    check(int(back) == 42, "conv underlying to named");
    Level lv = Level(2);
    check(int(lv) == 2, "conv int to enum");
    int total = int(u) + int(s) * 2;
    check(total == 88, "conv inside expressions");
    int8 k = int8(100);
    check(k == 100, "conv constant that fits");
    int8 j = int8(int16(1000) - 900);
    check(j == 100, "conv nested");
}
`},

	{dir: "conv_literals", src: `func main() {
    int8 a = 1;
    int16 b = 300;
    int64 c = 5000000000;
    int64[3] xs = [a, b, c];
    check(xs[0] == 1, "literal array widens an int8 element");
    check(xs[1] == 300, "literal array widens an int16 element");
    check(xs[2] == 5000000000, "literal array keeps an int64 element");
    let ys = [a, b];
    check(ys[1] == 300, "literal array takes the common element type");
    check(a in xs, "in widens the left operand");
    map[string]int64 m = {"one": a, "big": c};
    check(m["one"] == 1, "literal map widens a value");
    check(m["big"] == 5000000000, "literal map keeps an int64 value");
}
`},

	{dir: "const_decl", src: `const Limit = 100;
const Double = Limit * 2;
const Mask = 0xF0 | 0x0F;
const Big = 5000000000;
const int8 Small = 100;
const Neg = -5;
const uint8 Top = ~0;
const Shifted = 1 << 10;
const Mixed = Limit + 3 * 4;
const Cmp = Limit > 50;

enum Color {
    Red,
    Green,
    Blue
}

func main() {
    int8 a = Limit;
    int64 b = Limit;
    check(a == 100, "const untyped adapts to int8");
    check(b == 100, "const untyped adapts to int64");
    check(Double == 200, "const derived from another const");
    check(Mask == 255, "const bitwise expression");
    int64 big = Big;
    check(big == 5000000000, "const beyond 32 bits");
    check(Small == 100, "const with an explicit type");
    int n = Neg;
    check(n + 5 == 0, "const negative");
    check(Top == 255, "const complement of an unsigned type");
    check(Shifted == 1024, "const shift");
    check(Mixed == 112, "const precedence");
    check(Cmp, "const boolean");
    const Local = 7;
    check(Local + 1 == 8, "local const");
    check(Green == 1, "enum member is a const");
    int code = Blue;
    check(code == 2, "enum member converts to int");
}
`},

	{dir: "conv_struct", src: `struct Holder {
    int64 v;
}

func main() {
    int8 a = 100;
    Holder h = Holder{v: a};
    check(h.v == 100, "implicit struct field widening");
}
`},

	{dir: "conv_ternary", src: `func main() {
    int8 a = 100;
    int16 b = 300;
    int16 pick = true ? a : b;
    check(pick == 100, "implicit ternary branches widen");
    int16 other = false ? a : b;
    check(other == 300, "implicit ternary takes the common type");
}
`},

	{dir: "err_basic", src: `enum Kind {
    A,
    B
}

func f(int x) !int {
    if x == 0 {
        return Error(B);
    }
    return x * 2;
}

func main() {
    let v, e = f(5);
    check(e == null, "err success has no error");
    check(v == 10, "err success carries the value");
    let w, e2 = f(0);
    check(e2 != null, "err failure has an error");
    check(w == 0, "err failure payload is zero");
    check(e2 == B, "err failure is the member that was raised");
}
`},

	{dir: "err_propagate", src: `enum Kind {
    Neg,
    Other
}

func inner(int x) !int {
    if x < 0 {
        return Error(Neg);
    }
    return x;
}

func outer(int x) !int {
    let v = inner(x)!;
    return v + 1;
}

func twice(int x) !int {
    return inner(x)! + inner(x + 1)!;
}

func main() {
    let a, ea = outer(1);
    check(ea == null, "errprop success has no error");
    check(a == 2, "errprop success unwraps the value");
    let b, eb = outer(-1);
    check(eb == Neg, "errprop keeps the original error");
    check(b == 0, "errprop zeroes the payload");
    let c, ec = twice(1);
    check(ec == null, "errprop twice in one expression");
    check(c == 3, "errprop twice sums both values");
    let d, ed = twice(-1);
    check(ed == Neg, "errprop stops at the first failure");
}
`},

	{dir: "err_coalesce", src: `enum Kind {
    A,
    B
}

func f(int x) !int {
    if x == 0 {
        return Error(B);
    }
    return x * 2;
}

func main() {
    int a = f(5) ?? 99;
    check(a == 10, "errco default is skipped on success");
    int b = f(0) ?? 99;
    check(b == 99, "errco default is used on failure");
    int c = f(0) ?? |e| e.code;
    check(c == 1, "errco handler expression sees the error");
    int d = f(3) ?? |e| 7;
    check(d == 6, "errco handler is skipped on success");
    int g = (f(1) ?? 0) + (f(0) ?? 5);
    check(g == 7, "errco inside an expression");
}
`},

	{dir: "err_handler_block", src: `enum Kind {
    A
}

func f(int x) !int {
    if x == 0 {
        return Error(A);
    }
    return x * 2;
}

func g(int x) int {
    let v = f(x) ?? |e| {
        return -1;
    };
    return v + 100;
}

func h(int x) int {
    int out = 0;
    out = f(x) ?? |e| return -2;
    return out;
}

func main() {
    check(g(5) == 110, "errblock success continues");
    check(g(0) == -1, "errblock return leaves the function");
    check(h(4) == 8, "errblock return shorthand on success");
    check(h(0) == -2, "errblock return shorthand on failure");
}
`},

	{dir: "err_handler_loop", src: `enum Kind {
    A
}

func f(int x) !int {
    if x == 0 {
        return Error(A);
    }
    return x * 2;
}

func main() {
    int total = 0;
    for i in 0..3 {
        let v = f(i) ?? |e| {
            continue;
        };
        total += v;
    }
    check(total == 12, "errloop continue skips the failing iteration");
    int seen = 0;
    for i in 0..3 {
        let v = f(i) ?? |e| {
            break;
        };
        seen += 1;
    }
    check(seen == 0, "errloop break leaves the loop");
}
`},

	{dir: "err_void", src: `enum Kind {
    A
}

func v(int x) ! {
    if x == 0 {
        return Error(A);
    }
}

func caller(int x) ! {
    v(x)!;
}

func main() {
    let e1 = v(1);
    check(e1 == null, "errvoid success falls off the end");
    let e2 = v(0);
    check(e2 == A, "errvoid failure");
    let e3 = caller(1);
    check(e3 == null, "errvoid propagates success");
    let e4 = caller(0);
    check(e4 == A, "errvoid propagates the failure");
}
`},

	{dir: "err_multi", src: `enum Kind {
    A
}

func pair(int x) !(int, int) {
    if x == 0 {
        return Error(A);
    }
    return x, x * 10;
}

func sum(int x) !int {
    let a, b = pair(x)!;
    return a + b;
}

func main() {
    let a, b, e = pair(3);
    check(e == null, "errmulti success has no error");
    check(a == 3 && b == 30, "errmulti success carries both values");
    let c, d, e2 = pair(0);
    check(e2 == A, "errmulti failure");
    check(c == 0 && d == 0, "errmulti failure zeroes every value");
    let s, e3 = sum(2);
    check(e3 == null && s == 22, "errmulti propagate keeps both values");
    let t, e4 = sum(0);
    check(e4 == A && t == 0, "errmulti propagate stops on failure");
}
`},

	{dir: "err_domains", src: `enum First {
    Alpha
}

enum Second {
    Beta
}

func raise(int which) !int {
    if which == 0 {
        return Error(Alpha);
    }
    return Error(Beta);
}

func main() {
    let a, ea = raise(0);
    check(ea == Alpha, "errdomain matches its own member");
    check(ea != Beta, "errdomain different enums with the same code differ");
    let b, eb = raise(1);
    check(eb == Beta, "errdomain second enum");
    check(eb != Alpha, "errdomain does not match the other enum");
    check(ea.code == eb.code, "errdomain both codes are zero");
}
`},

	{dir: "err_switch", src: `enum Kind {
    A,
    B
}

func pick(int x) !int {
    if x == 0 {
        return Error(A);
    }
    if x < 0 {
        return Error(B);
    }
    return x;
}

func classify(int x) int {
    let v, e = pick(x);
    return switch e {
        case null => v,
        case A => 100,
        default => 200,
    };
}

func label(int x) int {
    let v, e = pick(x);
    int out = 0;
    switch e {
        case null:
            out = 1;
        case B:
            out = 2;
        default:
            out = 3;
    }
    return out;
}

func main() {
    check(classify(5) == 5, "errswitch expression on success");
    check(classify(0) == 100, "errswitch expression matches a member");
    check(classify(-1) == 200, "errswitch expression default");
    check(label(5) == 1, "errswitch statement on success");
    check(label(-1) == 2, "errswitch statement matches a member");
    check(label(0) == 3, "errswitch statement default");
}
`},

	{dir: "err_info", src: `enum Kind {
    A,
    B
}

func plain() !int {
    return Error(B);
}

func custom() !int {
    return Error(A, "something went wrong");
}

func main() {
    let x, e1 = plain();
    check(e1.message == "Kind.B", "errinfo default message is the member name");
    check(e1.code == 1, "errinfo code");
    check(e1.line > 0, "errinfo records a line");
    let y, e2 = custom();
    check(e2.message == "something went wrong", "errinfo custom message");
    check(e2.code == 0, "errinfo custom code");
}
`},

	{dir: "err_discard", src: `enum Kind {
    A
}

func f(int x) !int {
    if x == 0 {
        return Error(A);
    }
    return x;
}

func main() {
    let v, _ = f(4);
    check(v == 4, "errdiscard keeps the value");
    let _, e = f(0);
    check(e == A, "errdiscard keeps the error");
    let w, _ = f(0);
    check(w == 0, "errdiscard of a failure leaves zero");
}
`},

	{dir: "err_main", src: `enum Kind {
    A
}

func f(int x) !int {
    if x == 0 {
        return Error(A);
    }
    return x;
}

func main() ! {
    let v = f(3)!;
    check(v == 3, "errmain propagates inside main");
}
`},

	{dir: "err_compare_value", src: `enum Kind {
    A,
    B
}

func raise(int which) !int {
    if which == 0 {
        return Error(A);
    }
    return Error(B);
}

func isKind(Error e, Kind k) bool {
    return e == k;
}

func main() {
    let x, e = raise(0);
    Kind a = A;
    Kind b = B;
    check(e == a, "errvalue equals an enum variable");
    check(e != b, "errvalue differs from another enum variable");
    check(a == e, "errvalue enum variable on the left");
    check(isKind(e, A), "errvalue through a parameter");
    check(isKind(e, B) == false, "errvalue parameter mismatch");
    let y, other = raise(1);
    check(isKind(other, b), "errvalue second member");
}
`},
}
