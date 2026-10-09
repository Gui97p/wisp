package main

type manualCase struct {
	dir      string
	src      string
	expected []string
	abort    string
	x64Only  bool
	files    map[string]string
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

	{dir: "register_spill_left", src: `func main() {
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
    int q = (((((((((((a1 + a2) + a3) + a4) + a5) + a6) + a7) + a8) + a9) + a10) + a11) + a12);
    check(q == 78, "spill left nested sum");
}
`},

	{dir: "register_spill_sub", src: `func main() {
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
    int q = (a1 - (a2 - (a3 - (a4 - (a5 - (a6 - (a7 - (a8 - (a9 - (a10 - (a11 - a12)))))))))));
    check(q == -6, "spill right nested subtraction keeps order");
}
`},

	{dir: "register_spill_mixed", src: `func main() {
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
    int q = (a1 * (a2 + (a3 * (a4 + (a5 * (a6 + (a7 * (a8 + (a9 * (a10 + (a11 * a12)))))))))));
    check(q == 135134, "spill right nested mul and add");
}
`},

	{dir: "register_spill_index", src: `func main() {
    int[4] xs = [10, 20, 30, 40];
    int i1 = 1;
    int i2 = 2;
    int i3 = 3;
    int i4 = 0;
    int i5 = 1;
    int i6 = 2;
    int i7 = 3;
    int i8 = 0;
    int i9 = 1;
    int i10 = 2;
    int i11 = 3;
    int i12 = 0;
    int a = (xs[i1] + (xs[i2] + (xs[i3] + (xs[i4] + (xs[i5] + (xs[i6] + (xs[i7] + (xs[i8] + (xs[i9] + (xs[i10] + (xs[i11] + xs[i12])))))))))));
    check(a == 300, "spill with indexed elements pending");
    int b = ((i1 == 0 ? xs[i1] : 5) + ((i2 == 0 ? xs[i2] : 5) + ((i3 == 0 ? xs[i3] : 5) + ((i4 == 0 ? xs[i4] : 5) + ((i5 == 0 ? xs[i5] : 5) + ((i6 == 0 ? xs[i6] : 5) + ((i7 == 0 ? xs[i7] : 5) + ((i8 == 0 ? xs[i8] : 5) + ((i9 == 0 ? xs[i9] : 5) + ((i10 == 0 ? xs[i10] : 5) + ((i11 == 0 ? xs[i11] : 5) + (i12 == 0 ? xs[i12] : 5))))))))))));
    check(b == 75, "spill with ternaries and indexed elements pending");
    int[] part = xs[1:4];
    int d = (part[0] + (part[1] + (part[2] + (part[0] + (part[1] + (part[2] + (part[0] + (part[1] + (part[2] + (part[0] + (part[1] + part[2])))))))))));
    check(d == 360, "spill with span elements pending");
}
`},

	{dir: "register_spill_member", x64Only: true, src: `struct P {
    int x;
    int y;
}

func main() {
    P s = P{x: 3, y: 4};
    *P p = &s;
    int c = (p.x + (p.x + (p.x + (p.x + (p.x + (p.x + (p.x + (p.x + (p.x + (p.x + (p.x + p.y)))))))))));
    check(c == 37, "spill with members read through a pointer pending");
}
`},

	{dir: "register_spill_call", src: `func tag(string s, int n) int {
    return n + 1;
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
    int q = (a1 + (a2 + (a3 + (a4 + (a5 + (a6 + (a7 + (a8 + (a9 + tag("x", (a10 + (a11 + tag("y", a12)))))))))))));
    check(q == 80, "spill with calls and string args in the chain");
}
`},

	{dir: "agg_struct_basic", x64Only: true, src: `struct P {
    int x;
    int y;
}

struct Mixed {
    int8 a;
    int64 b;
    int16 c;
    bool d;
}

func main() {
    P p = P{x: 1, y: 2};
    check(p.x == 1, "struct literal first field");
    check(p.y == 2, "struct literal second field");
    p.x = 10;
    p.y += 5;
    check(p.x == 10, "struct field assignment");
    check(p.y == 7, "struct field compound assignment");
    P q = p;
    p.x = 99;
    check(q.x == 10, "struct copy is independent");
    Mixed m = Mixed{a: 3, b: 5000000000, c: 300, d: true};
    check(m.a == 3, "mixed int8 field");
    check(m.b == 5000000000, "mixed int64 field");
    check(m.c == 300, "mixed int16 field");
    check(m.d, "mixed bool field");
    m.c = 7;
    check(m.b == 5000000000, "writing one field leaves the neighbour alone");
    check(m.c == 7, "mixed field rewritten");
}
`},

	{dir: "agg_struct_zero", src: `struct P {
    int x;
    int y;
}

func main() {
    P blank;
    check(blank.x == 0, "uninitialised struct first field is zero");
    check(blank.y == 0, "uninitialised struct second field is zero");
}
`},

	{dir: "agg_struct_nested", src: `struct P {
    int x;
    int y;
}

struct Line {
    P a;
    P b;
    int tag;
}

func main() {
    Line l = Line{a: P{x: 1, y: 2}, b: P{x: 3, y: 4}, tag: 9};
    check(l.a.x == 1, "nested first x");
    check(l.b.y == 4, "nested second y");
    check(l.tag == 9, "field after nested structs");
    l.b.x = 30;
    check(l.b.x == 30, "nested field assignment");
    check(l.a.y == 2, "nested assignment leaves the sibling alone");
    P inner = l.b;
    check(inner.x == 30, "copy a nested struct out");
}
`},

	{dir: "agg_struct_abi", src: `struct P {
    int x;
    int y;
}

struct Big {
    int a;
    int b;
    int c;
}

func sum(P p) int {
    return p.x + p.y;
}

func make(int x, int y) P {
    return P{x: x, y: y};
}

func total(Big b) int {
    return b.a + b.b + b.c;
}

func build(int a) Big {
    return Big{a: a, b: a + 1, c: a + 2};
}

func main() {
    P p = P{x: 3, y: 4};
    check(sum(p) == 7, "struct passed by value in registers");
    P r = make(5, 6);
    check(r.x == 5, "struct returned in registers first");
    check(r.y == 6, "struct returned in registers second");
    Big b = Big{a: 1, b: 2, c: 3};
    check(total(b) == 6, "big struct passed by value");
    Big c = build(10);
    check(c.c == 12, "big struct returned through sret");
    check(sum(make(1, 2)) == 3, "struct returned and passed on");
}
`},

	{dir: "agg_index_const", src: `func main() {
    int[3] a;
    a[0] = 10;
    a[1] = 20;
    a[2] = 30;
    check(a[0] == 10, "constant index first");
    check(a[1] == 20, "constant index middle");
    check(a[2] == 30, "constant index last");
    a[1] += 5;
    check(a[1] == 25, "constant index compound");
    a[2]++;
    check(a[2] == 31, "constant index increment");
    int8[4] b;
    b[3] = 7;
    check(b[3] == 7, "constant index narrow element");
    check(b[2] == 0, "constant index neighbour stays zero");
}
`},

	{dir: "agg_array_basic", x64Only: true, src: `func main() {
    int[5] nums = [124, 512, 1, 5, 2];
    check(nums[0] == 124, "array literal first");
    check(nums[4] == 2, "array literal last");
    int i = 2;
    check(nums[i] == 1, "array read with a variable index");
    nums[0] = 999;
    nums[1]++;
    nums[2] += 10;
    check(nums[0] == 999, "array element assignment");
    check(nums[1] == 513, "array element increment");
    check(nums[2] == 11, "array element compound assignment");
    nums[i + 1] = 77;
    check(nums[3] == 77, "array write with a computed index");
    int[5] copy = nums;
    nums[0] = 0;
    check(copy[0] == 999, "array copy is independent");
    int total = 0;
    for k in 0..4 {
        total += copy[k];
    }
    check(total == 999 + 513 + 11 + 77 + 2, "array walked by a loop");
}
`},

	{dir: "agg_array_partial", src: `func main() {
    int[5] partial = [7, 8];
    check(partial[0] == 7, "partial array first");
    check(partial[1] == 8, "partial array second");
    check(partial[2] == 0, "partial array fills with zero");
    check(partial[4] == 0, "partial array last is zero");
    int8[6] small = [1, 2, 3];
    check(small[2] == 3, "narrow elements are packed");
    check(small[5] == 0, "narrow partial fills with zero");
    small[5] = 9;
    int8[4] bytes;
    check(bytes[0] == 0, "uninitialised array element is zero");
    check(bytes[3] == 0, "uninitialised array last element is zero");
    check(small[4] == 0, "narrow write leaves the neighbour alone");
    check(small[5] == 9, "narrow write lands");
}
`},

	{dir: "agg_array_of_struct", src: `struct P {
    int x;
    int y;
}

struct Bag {
    int[3] items;
    int count;
}

func main() {
    P[2] ps = [P{x: 1, y: 2}, P{x: 3, y: 4}];
    check(ps[0].x == 1, "array of struct first x");
    check(ps[1].y == 4, "array of struct second y");
    ps[1].x = 30;
    check(ps[1].x == 30, "array of struct field write");
    check(ps[0].y == 2, "array of struct write leaves the first alone");
    Bag b = Bag{items: [5, 6, 7], count: 3};
    check(b.items[1] == 6, "array field read");
    b.items[2] = 70;
    check(b.items[2] == 70, "array field write");
    check(b.count == 3, "field after an array field");
}
`},

	{dir: "agg_span_slice", x64Only: true, src: `func first(int[] xs) int {
    return xs[0];
}

func main() {
    int[5] nums = [10, 20, 30, 40, 50];
    int[] middle = nums[1:3];
    int[] head = nums[:2];
    int[] tail = nums[3:];
    check(len(middle) == 2, "slice length");
    check(middle[0] == 20, "slice first element");
    check(middle[1] == 30, "slice second element");
    check(len(head) == 2, "open start length");
    check(len(tail) == 2, "open end length");
    check(tail[1] == 50, "open end last element");
    nums[1] = 21;
    check(middle[0] == 21, "a slice is a view of the array");
    middle[1] = 31;
    check(nums[2] == 31, "writing through a slice changes the array");
    check(first(tail) == 40, "slice passed to a function");
}
`},

	{dir: "agg_null_pointer", x64Only: true, src: `func main() {
    *int p = null;
    check(p == null, "null pointer equals null");
    int v = 5;
    p = &v;
    check(p != null, "address is not null");
}
`},

	{dir: "ptr_basic", x64Only: true, src: `func swap(*int a, *int b) {
    int t = *a;
    *a = *b;
    *b = t;
}

func set(*int p, int v) {
    *p = v;
}

func main() {
    int v = 5;
    *int p = &v;
    check(*p == 5, "deref read");
    *p = 7;
    check(v == 7, "deref write reaches the variable");
    v = 9;
    check(*p == 9, "variable write is seen through the pointer");
    *p += 1;
    check(v == 10, "compound write through a pointer");
    int a = 1;
    int b = 2;
    swap(&a, &b);
    check(a == 2, "swap first");
    check(b == 1, "swap second");
    set(&a, 40);
    check(a == 40, "write through a pointer parameter");
    *int q = p;
    check(*q == 10, "pointer copy points at the same variable");
    **int pp = &p;
    check(**pp == 10, "double pointer read");
    **pp = 1;
    check(v == 1, "double pointer write");
    int8 first = 3;
    int8 second = 4;
    *int8 sp = &first;
    *sp = 100;
    check(first == 100, "narrow pointer write lands");
    check(second == 4, "narrow pointer write leaves the neighbour alone");
}
`},

	{dir: "ptr_elements", x64Only: true, src: `struct P {
    int x;
    int y;
}

func main() {
    int[3] a = [1, 2, 3];
    *int q = &a[1];
    *q = 20;
    check(a[1] == 20, "pointer to a constant index element");
    int i = 2;
    *int r = &a[i];
    *r = 30;
    check(a[2] == 30, "pointer to a variable index element");
    check(a[0] == 1, "element pointers leave the others alone");
    P s = P{x: 1, y: 2};
    *int py = &s.y;
    *py = 9;
    check(s.y == 9, "pointer to a struct field");
    check(s.x == 1, "field pointer leaves the other field alone");
    P[2] ps = [P{x: 1, y: 2}, P{x: 3, y: 4}];
    *int px = &ps[1].x;
    *px = 33;
    check(ps[1].x == 33, "pointer to a field of an array element");
    check(ps[0].x == 1, "that leaves the first element alone");
}
`},

	{dir: "ptr_struct", x64Only: true, src: `struct P {
    int x;
    int y;
}

struct Line {
    P a;
    P b;
}

func bump(*P p) {
    p.x += 1;
    p.y = p.x * 2;
}

func same(*P p) *P {
    return p;
}

func main() {
    P s = P{x: 1, y: 1};
    *P p = &s;
    check(p.x == 1, "member read through a pointer");
    p.x = 5;
    check(s.x == 5, "member write through a pointer");
    bump(p);
    check(s.x == 6, "pointer parameter member compound");
    check(s.y == 12, "pointer parameter member assignment");
    bump(&s);
    check(s.x == 7, "address passed directly");
    *P r = same(&s);
    r.y = 100;
    check(s.y == 100, "returned pointer reaches the original");
    P[2] ps = [P{x: 1, y: 2}, P{x: 3, y: 4}];
    *P e = &ps[1];
    e.y = 7;
    check(ps[1].y == 7, "pointer to an array element struct");
    check(ps[0].y == 2, "that leaves the first element alone");
    Line l = Line{a: P{x: 1, y: 2}, b: P{x: 3, y: 4}};
    *P pb = &l.b;
    pb.x = 40;
    check(l.b.x == 40, "pointer to a nested struct");
    check(l.a.x == 1, "nested pointer leaves the sibling alone");
    P copy = *p;
    p.x = 0;
    check(copy.x == 7, "dereferenced struct copy is independent");
}
`},

	{dir: "method_modules", files: map[string]string{"counter.wsp": `export struct Counter {
    int n;
}

export func (*Counter c) bump(int by) {
    c.n += by;
}

export func (*Counter c) get() int {
    return c.n;
}

func (*Counter c) doubled() int {
    return c.n * 2;
}

export func (*Counter c) twice() int {
    return c.doubled();
}

export func start(int n) Counter {
    return Counter{n: n};
}
`}, src: `import "counter";

struct Holder {
    counter.Counter inner;
}

func read(counter.Counter c) int {
    return c.get();
}

func make(int n) counter.Counter {
    return counter.start(n);
}

func main() {
    let c = counter.start(1);
    c.bump(2);
    check(c.get() == 3, "exported method of an imported struct");
    check(c.twice() == 6, "exported method that calls a private one");
    let d = counter.start(10);
    d.bump(5);
    check(d.get() == 15, "method on a variable of the imported type");
    check(c.get() == 3, "the other value is untouched");
    counter.Counter e = counter.Counter{n: 7};
    e.bump(1);
    check(e.get() == 8, "declared with the qualified type and literal");
    check(read(e) == 8, "qualified type as a parameter");
    let m = make(4);
    check(m.get() == 4, "qualified type as a return type");
    Holder h = Holder{inner: e};
    h.inner.bump(2);
    check(h.inner.get() == 10, "qualified type as a struct field");
    counter.Counter[2] pair = [counter.start(1), counter.start(2)];
    pair[1].bump(5);
    check(pair[1].get() == 7, "array of a qualified type");
    check(pair[0].get() == 1, "that leaves the other element alone");
}
`},

	{dir: "conv_nominal_string", src: `type Name string;

func greet(Name n) string {
    return string(n);
}

func main() {
    Name a = Name("ana");
    Name b = Name("ana");
    Name c = Name("bob");
    check(a == b, "nominal strings compare equal");
    check(a != c, "nominal strings compare different");
    check(greet(a) == "ana", "conversion back to string");
    string raw = string(c);
    check(raw == "bob", "string from a nominal string");
    int hits = 0;
    switch a {
        case Name("ana"):
            hits = 1;
        case Name("bob"):
            hits = 2;
        default:
            hits = 3;
    }
    check(hits == 1, "switch on a nominal string");
    Name[2] names = [Name("x"), Name("y")];
    check(names[1] == Name("y"), "array of nominal strings");
}
`},

	{dir: "enum_inferred", src: `enum Color {
    Red,
    Green,
    Blue = 10,
    Alpha,
}

struct Paint {
    Color c;
    int n;
}

const fav = Color.Blue;

func pick(bool b) Color {
    if b {
        return .Red;
    }
    return .Blue;
}

func code(Color c) int {
    switch c {
        case .Red:
            return 1;
        case .Green:
            return 2;
        default:
            return 3;
    }
}

func band(Color c) int {
    switch c {
        case Color.Red..Color.Green:
            return 1;
        case .Blue..Color.Alpha:
            return 2;
        default:
            return 3;
    }
}

func main() {
    Color a = .Green;
    check(a == Color.Green, "inferred member equals the qualified one");
    check(a == .Green, "comparison with an inferred member");
    check(.Green == a, "inferred member on the left");
    check(a != .Red, "inequality with an inferred member");
    a = .Blue;
    check(int(a) == 10, "assignment infers the enum");
    check(pick(true) == .Red, "return infers the enum");
    check(pick(false) == .Blue, "second return infers the enum");
    check(code(.Green) == 2, "argument infers the enum");
    check(code(Color.Red) == 1, "qualified argument");
    check(code(.Alpha) == 3, "switch default");
    check(band(.Green) == 1, "range of members");
    check(band(.Alpha) == 2, "range from an inferred start");
    check(band(a) == 2, "range with a variable");
    Paint p = Paint{c: .Alpha, n: 1};
    check(p.c == .Alpha, "struct field infers the enum");
    check(int(p.c) == 11, "implicit numbering after an explicit value");
    Color[3] cs = [.Red, Color.Green, .Blue];
    check(cs[0] == .Red, "array literal first");
    check(cs[1] == .Green, "array literal mixed");
    check(cs[2] == .Blue, "array literal inferred");
    check(fav == Color.Blue, "const holding an enum member");
    check(int(fav) == 10, "const enum converts to int");
}
`},

	{dir: "ternary_falsy", src: `func pickBool(bool c) bool {
    return c ? false : true;
}

func main() {
    bool yes = true;
    bool no = false;
    bool a = yes ? false : true;
    check(a == false, "ternary yields false from the then branch");
    bool b = no ? false : true;
    check(b == true, "ternary yields true from the else branch");
    check(pickBool(true) == false, "ternary with false in a return");
    check(pickBool(false) == true, "ternary with true in a return");
    int n = (yes ? 1 : 2) + (no ? 10 : 20);
    check(n == 21, "ternaries joined by an operator");
    int m = (yes ? 1 : 2) * (yes ? 3 : 4) + (no ? 5 : 6);
    check(m == 9, "ternaries inside arithmetic");
}
`},

	{dir: "type_modules", files: map[string]string{"units.wsp": `export type Meters int;

export enum Level {
    Low,
    High,
}

export func twice(Meters m) Meters {
    return Meters(int(m) * 2);
}

export func isHigh(Level l) bool {
    return l == .High;
}
`}, src: `import "units";

func grow(units.Meters m) units.Meters {
    return units.twice(m);
}

func main() {
    units.Meters a = units.Meters(21);
    check(int(a) == 21, "conversion to an imported nominal type");
    units.Meters b = grow(a);
    check(int(b) == 42, "imported nominal type through a function");
    int raw = int(units.twice(units.Meters(4)));
    check(raw == 8, "conversion chain across a module boundary");
    units.Level lv = units.Level.High;
    check(lv == units.Level.High, "qualified member of an imported enum");
    check(units.isHigh(lv), "imported enum through a function");
    check(units.isHigh(.High), "inferred member for an imported enum parameter");
    check(!units.isHigh(.Low), "the other member");
    units.Level other = .Low;
    check(other != lv, "inferred member assigned to an imported enum type");
}
`},

	{dir: "ptr_double", x64Only: true, src: `struct P {
    int x;
    int y;
}

func setAll(**P pp, int v) {
    (*pp).x = v;
    (**pp).y = v + 1;
}

func main() {
    P s = P{x: 1, y: 2};
    *P p = &s;
    **P pp = &p;
    check((*pp).x == 1, "member through a double pointer");
    check((**pp).y == 2, "member of a fully dereferenced pointer");
    (*pp).x = 10;
    check(s.x == 10, "write through a double pointer");
    (**pp).y += 5;
    check(s.y == 7, "compound write of a fully dereferenced pointer");
    setAll(pp, 40);
    check(s.x == 40, "double pointer parameter first field");
    check(s.y == 41, "double pointer parameter second field");
    P other = P{x: 100, y: 200};
    *pp = &other;
    check(p.x == 100, "writing the inner pointer redirects the pointer");
    check(s.x == 40, "the old target is untouched");
}
`},

	{dir: "ptr_null", x64Only: true, src: `struct Node {
    *int p;
    int tag;
}

func pick(bool yes, *int p) *int {
    if yes {
        return p;
    }
    return null;
}

func isNull(*int p) bool {
    return p == null;
}

func main() {
    int v = 1;
    *int a = null;
    check(a == null, "null pointer equals null");
    check(null == a, "null on the left");
    a = &v;
    check(a != null, "address is not null");
    check(null != a, "null on the left, not equal");
    check(isNull(null), "null passed as an argument");
    check(!isNull(&v), "address passed as an argument");
    Node n = Node{tag: 5};
    check(n.p == null, "omitted pointer field is null");
    n.p = &v;
    check(n.p != null, "assigned pointer field is not null");
    *n.p = 8;
    check(v == 8, "write through a pointer field");
    check(pick(false, &v) == null, "returned null");
    check(pick(true, &v) != null, "returned pointer");
    a = null;
    check(a == null, "null assigned back");
}
`},

	{dir: "method_basic", src: `struct Counter {
    int n;
}

func (*Counter c) bump(int by) {
    c.n += by;
}

func (*Counter c) get() int {
    return c.n;
}

func (*Counter c) twice() {
    c.bump(1);
    c.bump(1);
}

func (*Counter c) pair() (int, int) {
    return c.n, c.n * 2;
}

struct Pair {
    Counter a;
    Counter b;
}

func main() {
    Counter c = Counter{n: 1};
    c.bump(2);
    check(c.get() == 3, "method changes the receiver");
    c.twice();
    check(c.n == 5, "method calling methods");
    let x, y = c.pair();
    check(x == 5, "method multiple return first");
    check(y == 10, "method multiple return second");
    Counter[2] cs = [Counter{n: 1}, Counter{n: 2}];
    cs[1].bump(5);
    check(cs[1].n == 7, "method on an array element");
    check(cs[0].n == 1, "that leaves the other element alone");
    Pair p = Pair{a: Counter{n: 1}, b: Counter{n: 2}};
    p.b.bump(10);
    check(p.b.n == 12, "method on a nested field");
    check(p.a.n == 1, "that leaves the sibling alone");
}
`},

	{dir: "method_pointer", x64Only: true, src: `struct Counter {
    int n;
}

func (*Counter c) bump(int by) {
    c.n += by;
}

func (*Counter c) get() int {
    return c.n;
}

func (Counter c) peek() int {
    return c.n;
}

func (Counter c) reset() int {
    c.n = 0;
    return c.n;
}

func main() {
    Counter c = Counter{n: 1};
    *Counter p = &c;
    p.bump(10);
    check(c.n == 11, "method called through a pointer");
    check(p.get() == 11, "getter through a pointer");
    check(c.peek() == 11, "value receiver read");
    check(p.peek() == 11, "value receiver through a pointer");
    check(c.reset() == 0, "value receiver sees its own change");
    check(c.n == 11, "value receiver works on a copy");
}
`},

	{dir: "string_eq", src: `func main() {
    check("abc" == "abc", "equal strings");
    check("abc" != "abd", "same length, different last byte");
    check("abc" != "ab", "different length");
    check("ab" != "abc", "prefix is not equal");
    check("" == "", "empty strings are equal");
    check("a" != "", "empty differs from non-empty");
    string s = "hello";
    string t = "hel";
    check(s != t, "variables of different length");
    check(s[0:3] == t, "a slice compares equal to its text");
    check(s[1:3] == "el", "a middle slice");
    check(s[2:2] == "", "an empty slice is the empty string");
    string[2] arr = ["x", "y"];
    check(arr[0] == "x", "array element equals");
    check(arr[1] != "x", "array element differs");
    string u = "hello";
    check(s == u, "equal contents in two variables");
    check(!(s == t), "not of equality");
}
`},

	{dir: "switch_string", src: `func name(string s) int {
    int r = 0;
    switch s {
        case "a":
            r = 1;
        case "bc":
            r = 2;
        case "":
            r = 3;
        default:
            r = 4;
    }
    return r;
}

func main() {
    check(name("a") == 1, "switch string first");
    check(name("bc") == 2, "switch string second");
    check(name("") == 3, "switch on the empty string");
    check(name("zz") == 4, "switch string default");
    check(name("b") == 4, "a prefix of a case is not a match");
    check(name("bcd") == 4, "an extension of a case is not a match");
}
`},

	{dir: "for_in_collection", src: `func main() {
    int[4] a = [10, 20, 30, 40];
    int total = 0;
    int idx = 0;
    for i, v in a {
        total += v;
        idx += i;
    }
    check(total == 100, "for in array sums the values");
    check(idx == 6, "for in array indexes");
    int count = 0;
    for i in a {
        count += 1;
    }
    check(count == 4, "single name walks the indexes");
    int[] part = a[1:3];
    int sum = 0;
    for v in part {
        sum += 1;
    }
    check(sum == 2, "for in span visits the span length");
    int psum = 0;
    for i, v in part {
        psum += v;
    }
    check(psum == 50, "for in span values");
    string word = "abc";
    int chars = 0;
    int last = 0;
    for i, c in word {
        chars += 1;
        last = i;
    }
    check(chars == 3, "for in string visits every byte");
    check(last == 2, "for in string last index");
    int empty = 0;
    int[] none = a[2:2];
    for i, v in none {
        empty += 1;
    }
    check(empty == 0, "for in an empty span runs zero times");
    int stop = 0;
    for i, v in a {
        if v == 30 {
            break;
        }
        stop += 1;
    }
    check(stop == 2, "break leaves a for in");
    int skipped = 0;
    for i, v in a {
        if v == 20 {
            continue;
        }
        skipped += v;
    }
    check(skipped == 80, "continue skips an element");
    int pairs = 0;
    for i, v in a {
        for j, w in a {
            pairs += 1;
        }
    }
    check(pairs == 16, "nested for in");
    check(a[0] == 10, "the collection is not modified");
}
`},

	{dir: "in_scalar", src: `enum Color {
    Red,
    Green,
    Blue,
}

func main() {
    int[4] a = [10, 20, 30, 40];
    check(30 in a, "in finds a middle element");
    check(10 in a, "in finds the first element");
    check(40 in a, "in finds the last element");
    check(!(50 in a), "in does not find a missing value");
    int8 small = 20;
    check(small in a, "in widens the left operand");
    int[] part = a[1:3];
    check(20 in part, "in finds an element of a span");
    check(!(10 in part), "in is limited to the span");
    check(!(40 in part), "in stops at the end of the span");
    int[] none = a[2:2];
    check(!(10 in none), "in an empty span finds nothing");
    string word = "hello";
    check('e' in word, "in finds a char in a string");
    check('h' in word, "in finds the first char");
    check('o' in word, "in finds the last char");
    check(!('z' in word), "in does not find a missing char");
    int8[3] bytes = [1, 2, 3];
    check(2 in bytes, "in over narrow elements");
    check(!(4 in bytes), "in over narrow elements, missing");
    bool[2] flags = [true, true];
    check(true in flags, "in over bool elements");
    check(!(false in flags), "in over bool elements, missing");
    Color[2] cols = [Color.Red, Color.Blue];
    check(Color.Blue in cols, "in over enum elements");
    check(!(Color.Green in cols), "in over enum elements, missing");
    int target = 40;
    check(target in a, "in with a variable on the left");
    check((10 + 20) in a, "in with an expression on the left");
}
`},

	{dir: "conv_char_string", src: `func main() {
    char c = 'a';
    string s = string(c);
    check(s == "a", "char converts to a one byte string");
    check(len(s) == 1, "the string has length one");
    check(string('z') == "z", "converted char literal");
    string word = "hey";
    string first = string(word[0]);
    check(first == "h", "indexing a string then converting");
}
`},

	{dir: "in_string", src: `func main() {
    string word = "hello";
    check("ell" in word, "in finds a substring");
    check("hello" in word, "in finds the whole string");
    check("h" in word, "in finds a one byte substring");
    check("lo" in word, "in finds a suffix");
    check(!("elx" in word), "in does not find a near substring");
    check(!("hellos" in word), "in does not find a longer string");
    check("" in word, "the empty string is in every string");
    check("" in "", "the empty string is in the empty string");
    string[3] names = ["ann", "bob", "cy"];
    check("bob" in names, "in finds a string element");
    check("ann" in names, "in finds the first string element");
    check("cy" in names, "in finds the last string element");
    check(!("zed" in names), "in does not find a missing string");
    check(!("bo" in names), "a prefix of an element is not an element");
    check(!("bobby" in names), "an extension of an element is not an element");
    string[] part = names[1:3];
    check("cy" in part, "in finds a string in a span");
    check(!("ann" in part), "in is limited to the span of strings");
    string[] none = names[1:1];
    check(!("bob" in none), "in an empty span of strings finds nothing");
    string first = names[0];
    check(first in names, "in with a string variable");
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
    Color col = Color.Green;
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
    check(Color.Green == 1, "enum member is a const");
    int code = Color.Blue;
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
        return Error(Kind.B);
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
    check(e2 == Kind.B, "err failure is the member that was raised");
}
`},

	{dir: "err_propagate", src: `enum Kind {
    Neg,
    Other
}

func inner(int x) !int {
    if x < 0 {
        return Error(Kind.Neg);
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
    check(eb == Kind.Neg, "errprop keeps the original error");
    check(b == 0, "errprop zeroes the payload");
    let c, ec = twice(1);
    check(ec == null, "errprop twice in one expression");
    check(c == 3, "errprop twice sums both values");
    let d, ed = twice(-1);
    check(ed == Kind.Neg, "errprop stops at the first failure");
}
`},

	{dir: "err_coalesce", src: `enum Kind {
    A,
    B
}

func f(int x) !int {
    if x == 0 {
        return Error(Kind.B);
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
        return Error(Kind.A);
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
        return Error(Kind.A);
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
        return Error(Kind.A);
    }
}

func caller(int x) ! {
    v(x)!;
}

func main() {
    let e1 = v(1);
    check(e1 == null, "errvoid success falls off the end");
    let e2 = v(0);
    check(e2 == Kind.A, "errvoid failure");
    let e3 = caller(1);
    check(e3 == null, "errvoid propagates success");
    let e4 = caller(0);
    check(e4 == Kind.A, "errvoid propagates the failure");
}
`},

	{dir: "err_multi", src: `enum Kind {
    A
}

func pair(int x) !(int, int) {
    if x == 0 {
        return Error(Kind.A);
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
    check(e2 == Kind.A, "errmulti failure");
    check(c == 0 && d == 0, "errmulti failure zeroes every value");
    let s, e3 = sum(2);
    check(e3 == null && s == 22, "errmulti propagate keeps both values");
    let t, e4 = sum(0);
    check(e4 == Kind.A && t == 0, "errmulti propagate stops on failure");
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
        return Error(First.Alpha);
    }
    return Error(Second.Beta);
}

func main() {
    let a, ea = raise(0);
    check(ea == First.Alpha, "errdomain matches its own member");
    check(ea != Second.Beta, "errdomain different enums with the same code differ");
    let b, eb = raise(1);
    check(eb == Second.Beta, "errdomain second enum");
    check(eb != First.Alpha, "errdomain does not match the other enum");
    check(ea.code == eb.code, "errdomain both codes are zero");
}
`},

	{dir: "err_switch", src: `enum Kind {
    A,
    B
}

func pick(int x) !int {
    if x == 0 {
        return Error(Kind.A);
    }
    if x < 0 {
        return Error(Kind.B);
    }
    return x;
}

func classify(int x) int {
    let v, e = pick(x);
    return switch e {
        case null => v,
        case Kind.A => 100,
        default => 200,
    };
}

func label(int x) int {
    let v, e = pick(x);
    int out = 0;
    switch e {
        case null:
            out = 1;
        case Kind.B:
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
    return Error(Kind.B);
}

func custom() !int {
    return Error(Kind.A, "something went wrong");
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
        return Error(Kind.A);
    }
    return x;
}

func main() {
    let v, _ = f(4);
    check(v == 4, "errdiscard keeps the value");
    let _, e = f(0);
    check(e == Kind.A, "errdiscard keeps the error");
    let w, _ = f(0);
    check(w == 0, "errdiscard of a failure leaves zero");
}
`},

	{dir: "err_main", src: `enum Kind {
    A
}

func f(int x) !int {
    if x == 0 {
        return Error(Kind.A);
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
        return Error(Kind.A);
    }
    return Error(Kind.B);
}

func isKind(Error e, Kind k) bool {
    return e == k;
}

func main() {
    let x, e = raise(0);
    Kind a = Kind.A;
    Kind b = Kind.B;
    check(e == a, "errvalue equals an enum variable");
    check(e != b, "errvalue differs from another enum variable");
    check(a == e, "errvalue enum variable on the left");
    check(isKind(e, Kind.A), "errvalue through a parameter");
    check(isKind(e, Kind.B) == false, "errvalue parameter mismatch");
    let y, other = raise(1);
    check(isKind(other, b), "errvalue second member");
}
`},

	{dir: "divmod_registers", src: `func idiv(int a, int b) int {
    return a / b;
}

func main() {
    int a1 = 1;
    int a2 = 2;
    int b = 20;
    int c = 4;
    check((a1 + a2) + (b / c) == 8, "divreg ax busy while dividing");
    check((a1 + a2) + (b % 7) == 9, "divreg ax busy while taking the remainder");
    check((b / c) + (b / c) == 10, "divreg two divisions in one expression");
    check((a1 + b) / (a2 + c) == 3, "divreg operands are registers");
    check((b + c) % (a1 + a2) == 0, "divreg remainder of registers");
    check(b / c / a2 == 2, "divreg chained divisions");
    check((a1 + a2) + ((b / c) + ((a1 + a2) + (c / a2))) == 13, "divreg nested with live registers");
    check(idiv(100, 7) + idiv(9, 2) == 18, "divreg through calls");
    int8 s = 100;
    int8 t = 7;
    check(s / t == 14, "divreg int8 quotient");
    check(s % t == 2, "divreg int8 remainder");
    uint8 u = 200;
    uint8 v = 7;
    check(u / v == 28, "divreg uint8 quotient");
    check(u % v == 4, "divreg uint8 remainder");
    uint64 big = 18446744073709551615;
    uint64 three = 3;
    check(big / three == 6148914691236517205, "divreg uint64 high values");
    check(big % three == 0, "divreg uint64 remainder");
    int x = 100;
    x /= c;
    check(x == 25, "divreg compound division");
    x %= 7;
    check(x == 4, "divreg compound remainder");
}
`},

	{dir: "abort_div_zero", src: `func main() {
    int a = 10;
    int z = 0;
    check(a / 2 == 5, "abortdiv before the division by zero");
    int r = a / z;
    check(true, "abortdiv never printed");
}
`, expected: []string{"abortdiv before the division by zero"}, abort: "panic: integer divide by zero"},

	{dir: "abort_mod_zero", src: `func main() {
    int a = 10;
    int z = 0;
    check(a % 3 == 1, "abortmod before the remainder by zero");
    int r = a % z;
    check(true, "abortmod never printed");
}
`, expected: []string{"abortmod before the remainder by zero"}, abort: "panic: integer divide by zero"},

	{dir: "abort_div_zero_int8", src: `func main() {
    int8 a = 100;
    int8 z = 0;
    check(a / 2 == 50, "abortdiv8 before the division by zero");
    int8 r = a / z;
    check(true, "abortdiv8 never printed");
}
`, expected: []string{"abortdiv8 before the division by zero"}, abort: "panic: integer divide by zero"},

	{dir: "abort_div_zero_uint64", src: `func main() {
    uint64 a = 18446744073709551615;
    uint64 z = 0;
    check(a / 2 == 9223372036854775807, "abortdivu64 before the division by zero");
    uint64 r = a / z;
    check(true, "abortdivu64 never printed");
}
`, expected: []string{"abortdivu64 before the division by zero"}, abort: "panic: integer divide by zero"},

	{dir: "abort_div_zero_compound", src: `func main() {
    int a = 10;
    int z = 0;
    a /= 2;
    check(a == 5, "abortdivc before the division by zero");
    a /= z;
    check(true, "abortdivc never printed");
}
`, expected: []string{"abortdivc before the division by zero"}, abort: "panic: integer divide by zero"},

	{dir: "abort_div_zero_expression", src: `func main() {
    int a = 10;
    int b = 5;
    check((a + b) / (a - 5) == 3, "abortdive before the division by zero");
    int r = (a + b) / (a - a);
    check(true, "abortdive never printed");
}
`, expected: []string{"abortdive before the division by zero"}, abort: "panic: integer divide by zero"},

	{dir: "abort_panic_builtin", src: `func main() {
    check(true, "abortpanic before the panic");
    panic("custom message");
    check(true, "abortpanic never printed");
}
`, expected: []string{"abortpanic before the panic"}, abort: "panic: custom message"},

	{dir: "div_min_by_minus_one", src: `func main() {
    int8 min8 = 0;
    min8 -= 100;
    min8 -= 28;
    int8 neg8 = 0;
    neg8 -= 1;
    check(min8 / neg8 == min8, "minneg int8 quotient wraps to min");
    check(min8 % neg8 == 0, "minneg int8 remainder is zero");
    int32 min32 = 0;
    min32 -= 2147483647;
    min32 -= 1;
    int32 neg32 = 0;
    neg32 -= 1;
    check(min32 / neg32 == min32, "minneg int32 quotient wraps to min");
    check(min32 % neg32 == 0, "minneg int32 remainder is zero");
    int64 min64 = 0;
    min64 -= 9223372036854775807;
    min64 -= 1;
    int64 neg64 = 0;
    neg64 -= 1;
    check(min64 / neg64 == min64, "minneg int64 quotient wraps to min");
    check(min64 % neg64 == 0, "minneg int64 remainder is zero");
    int64 seven = 7;
    check(seven / neg64 == 0 - 7, "minneg ordinary value by minus one");
}
`},

	{dir: "flow_range_directions", src: `func main() {
    int up = 0;
    for i in 1..4 {
        up = up * 10 + i;
    }
    check(up == 1234, "range ascending visits every value");
    int down = 0;
    for i in 4..1 {
        down = down * 10 + i;
    }
    check(down == 4321, "range descending counts down");
    int once = 0;
    for i in 5..5 {
        once += 1;
    }
    check(once == 1, "range with equal bounds runs once");
    int stepUp = 0;
    for i in 1..10:4 {
        stepUp = stepUp * 100 + i;
    }
    check(stepUp == 10509, "range ascending with a step");
    int stepDown = 0;
    for i in 10..1:3 {
        stepDown = stepDown * 100 + i;
    }
    check(stepDown == 10070401, "range descending with a step");
    int skip = 0;
    for i in 10..1:20 {
        skip += 1;
    }
    check(skip == 1, "range step larger than the distance visits only the start");
}
`},

	{dir: "flow_range_evaluation", src: `func bound(int v) int {
    return v;
}

func main() {
    int n = 5;
    int seen = 0;
    for i in 1..n {
        seen += 1;
        n = 2;
    }
    check(seen == 5, "range end is read once");
    int calls = 0;
    for i in 1..bound(3) {
        calls += 1;
    }
    check(calls == 3, "range with a call as its end runs the right number of times");
    int step = 2;
    int stepped = 0;
    for i in 1..9:step {
        stepped += 1;
        step = 100;
    }
    check(stepped == 5, "range step is read once");
    int edits = 0;
    for j in 1..3 {
        j += 1;
        edits += 1;
    }
    check(edits == 2, "changing the loop variable in the body changes the iteration");
    int a = 3;
    int b = 1;
    int dir = 0;
    for i in a..b {
        dir += 1;
    }
    check(dir == 3, "range direction comes from variables");
}
`},

	{dir: "abort_range_step_zero", src: `func main() {
    int step = 0;
    check(true, "abortstep before the loop");
    for i in 1..5:step {
        check(true, "abortstep never printed");
    }
}
`, expected: []string{"abortstep before the loop"}, abort: "panic: range step must be positive"},

	{dir: "abort_index_array", x64Only: true, src: `func main() {
    int[5] nums = [1, 2, 3, 4, 5];
    int i = 4;
    check(nums[i] == 5, "abortidx last element is readable");
    i = 5;
    check(true, "abortidx before the bad read");
    int x = nums[i];
    check(false, "abortidx never printed");
}
`, expected: []string{"abortidx last element is readable", "abortidx before the bad read"}, abort: "panic: index out of range"},

	{dir: "abort_index_negative", x64Only: true, src: `func main() {
    int[5] nums = [1, 2, 3, 4, 5];
    int i = 0;
    i -= 1;
    check(true, "abortneg before the bad read");
    int x = nums[i];
    check(false, "abortneg never printed");
}
`, expected: []string{"abortneg before the bad read"}, abort: "panic: index out of range"},

	{dir: "abort_index_write", x64Only: true, src: `func main() {
    int[3] nums = [1, 2, 3];
    int i = 3;
    check(true, "abortwrite before the bad write");
    nums[i] = 9;
    check(false, "abortwrite never printed");
}
`, expected: []string{"abortwrite before the bad write"}, abort: "panic: index out of range"},

	{dir: "abort_index_span", x64Only: true, src: `func main() {
    int[5] nums = [1, 2, 3, 4, 5];
    int[] part = nums[1:3];
    int i = 2;
    check(part[1] == 3, "abortspan last element is readable");
    check(true, "abortspan before the bad read");
    int x = part[i];
    check(false, "abortspan never printed");
}
`, expected: []string{"abortspan last element is readable", "abortspan before the bad read"}, abort: "panic: index out of range"},

	{dir: "abort_slice_end", x64Only: true, src: `func main() {
    int[5] nums = [1, 2, 3, 4, 5];
    int end = 6;
    check(true, "abortslice before the bad slice");
    int[] part = nums[1:end];
    check(false, "abortslice never printed");
}
`, expected: []string{"abortslice before the bad slice"}, abort: "panic: slice out of range"},

	{dir: "abort_slice_order", x64Only: true, src: `func main() {
    int[5] nums = [1, 2, 3, 4, 5];
    int start = 3;
    int end = 2;
    check(true, "abortorder before the bad slice");
    int[] part = nums[start:end];
    check(false, "abortorder never printed");
}
`, expected: []string{"abortorder before the bad slice"}, abort: "panic: slice out of range"},

	{dir: "flow_switch_in_loop", src: `func main() {
    int hits = 0;
    for i in 1..5 {
        switch i {
            case 2:
                continue;
            case 4:
                break;
            default:
                hits += 10;
        }
        hits += 1;
    }
    check(hits == 34, "switchloop break leaves the switch and continue skips the rest of the iteration");
    int outer = 0;
    :rows for i in 1..3 {
        for j in 1..3 {
            switch j {
                case 2:
                    continue rows;
                default:
                    outer += 1;
            }
        }
    }
    check(outer == 3, "switchloop continue with a label crosses the switch");
}
`},

	{dir: "flow_range_types", src: `func main() {
    int8 lo = 2;
    int16 hi = 5;
    int acc = 0;
    for i in lo..hi {
        acc = acc * 10 + i;
    }
    check(acc == 2345, "rangetypes bounds of narrower types widen to int");
    uint8 count = 3;
    int turns = 0;
    for count {
        turns += 1;
    }
    check(turns == 3, "rangetypes iteration count of a narrow type");
    int8 step = 3;
    int seen = 0;
    for i in 1..10:step {
        seen += 1;
    }
    check(seen == 4, "rangetypes step of a narrow type");
}
`},
}
