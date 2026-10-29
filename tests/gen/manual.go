package main

type manualCase struct {
	dir      string
	src      string
	expected []string
	abort    string
	exit     int
	x64Only  bool
	luaOnly  bool
	libc     bool
	cmain    bool
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
    float a = 2.5;
    float64 b = a;
    float c = b * 2.0;
    check(c == 5.0, "float is another name for float64");
    let inferred = 1.5;
    float e = inferred;
    check(e == 1.5, "an untyped float literal infers float");
    float32 s = 1.5;
    float widened = s;
    check(widened == 1.5, "float32 widens to float");
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

	{dir: "array_to_pointer", src: `func fill(*char p, int n, char c) {
    for i in 0..n - 1 {
        (p + i)@ = c;
    }
}

func first(*void p) *char {
    return (*char)(p);
}

func main() {
    char[8] buf;
    fill((*char)(buf), 8, 'a');
    check(buf[0] == 'a' && buf[7] == 'a', "an array converts to a pointer to its first element");

    char[] view = buf[2:6];
    fill((*char)(view), 4, 'z');
    check(buf[1] == 'a' && buf[2] == 'z' && buf[5] == 'z' && buf[6] == 'a', "a span points into the array it views");

    char[] empty = buf[3:3];
    *char none = (*char)(empty);
    check(none == (*char)(buf) + 3, "the pointer of an empty span is still its start");

    int[3] nums;
    *void raw = (*void)(nums);
    *int back = (*int)(raw);
    back@ = 7;
    check(nums[0] == 7, "an array converts to *void and back");

    check(first((*void)(buf))@ == 'a', "a *void from an array reads as bytes");
}
`, expected: []string{"an array converts to a pointer to its first element", "a span points into the array it views", "the pointer of an empty span is still its start", "an array converts to *void and back", "a *void from an array reads as bytes"}},

	{dir: "register_spill_member", src: `struct P {
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

	{dir: "agg_struct_basic", src: `struct P {
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

	{dir: "agg_array_basic", src: `func main() {
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

	{dir: "agg_span_slice", src: `func first(int[] xs) int {
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

	{dir: "agg_null_pointer", src: `func main() {
    *int p = null;
    check(p == null, "null pointer equals null");
    int v = 5;
    p = &v;
    check(p != null, "address is not null");
}
`},

	{dir: "ptr_basic", src: `func swap(*int a, *int b) {
    int t = a@;
    a@ = b@;
    b@ = t;
}

func set(*int p, int v) {
    p@ = v;
}

func main() {
    int v = 5;
    *int p = &v;
    check(p@ == 5, "deref read");
    p@ = 7;
    check(v == 7, "deref write reaches the variable");
    v = 9;
    check(p@ == 9, "variable write is seen through the pointer");
    p@ += 1;
    check(v == 10, "compound write through a pointer");
    int a = 1;
    int b = 2;
    swap(&a, &b);
    check(a == 2, "swap first");
    check(b == 1, "swap second");
    set(&a, 40);
    check(a == 40, "write through a pointer parameter");
    *int q = p;
    check(q@ == 10, "pointer copy points at the same variable");
    **int pp = &p;
    check(pp@@ == 10, "double pointer read");
    pp@@ = 1;
    check(v == 1, "double pointer write");
    int8 first = 3;
    int8 second = 4;
    *int8 sp = &first;
    sp@ = 100;
    check(first == 100, "narrow pointer write lands");
    check(second == 4, "narrow pointer write leaves the neighbour alone");
}
`},

	{dir: "ptr_elements", src: `struct P {
    int x;
    int y;
}

func main() {
    int[3] a = [1, 2, 3];
    *int q = &a[1];
    q@ = 20;
    check(a[1] == 20, "pointer to a constant index element");
    int i = 2;
    *int r = &a[i];
    r@ = 30;
    check(a[2] == 30, "pointer to a variable index element");
    check(a[0] == 1, "element pointers leave the others alone");
    P s = P{x: 1, y: 2};
    *int py = &s.y;
    py@ = 9;
    check(s.y == 9, "pointer to a struct field");
    check(s.x == 1, "field pointer leaves the other field alone");
    P[2] ps = [P{x: 1, y: 2}, P{x: 3, y: 4}];
    *int px = &ps[1].x;
    px@ = 33;
    check(ps[1].x == 33, "pointer to a field of an array element");
    check(ps[0].x == 1, "that leaves the first element alone");
}
`},

	{dir: "ptr_struct", src: `struct P {
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
    P copy = p@;
    p.x = 0;
    check(copy.x == 7, "dereferenced struct copy is independent");
}
`},

	{dir: "internal_package", files: map[string]string{"shapes/shapes.wsp": `import "shapes/sides";
import "shapes/deep";

public func boxArea(int w) int {
    sides.Box b = sides.Box{w: w, scale: 2};
    return b.w * b.scale;
}

public func corners(int n) int {
    return deep.twice(sides.count(n)) + sides.hidden() - 1;
}
`, "shapes/deep/deep.wsp": `shared func twice(int n) int {
    return n * 2;
}
`, "shapes/sides.wsp": `public func count(int n) int {
    return n + hidden();
}

internal func hidden() int {
    return 1;
}

public struct Box {
    int w;
    internal int scale;
}

internal func (*Box b) area() int {
    return b.w * b.w;
}

public func grown(int w) int {
    Box b = Box{w: w};
    return b.area() + 1;
}
`}, src: `import "shapes";

func main() {
    check(shapes.corners(3) == 8, "a package entry reaches its internal files");
    check(shapes.count(3) == 4, "a package exposes what its files make public");
    check(shapes.grown(3) == 10, "an internal method works inside its package");
    check(shapes.boxArea(3) == 6, "an internal field is used by the files of its package");
}
`, expected: []string{"a package entry reaches its internal files", "a package exposes what its files make public", "an internal method works inside its package", "an internal field is used by the files of its package"}},
	{dir: "array_to_slice", x64Only: true, src: `struct Box {
    int[3] items;
}

func total(int[] xs) int {
    int sum;
    for _, x in xs {
        sum += x;
    }
    return sum;
}

func bump(int[] xs) {
    xs[0] += 100;
}

func main() {
    int[4] nums = [1, 2, 3, 4];
    check(total(nums) == 10, "an array variable passes as a slice");

    bump(nums);
    check(nums[0] == 101, "the slice is a view of the array");

    int[] view = nums;
    view[1] = 20;
    check(nums[1] == 20, "an array initialises a slice variable");

    int[] other;
    other = nums;
    other[2] = 30;
    check(nums[2] == 30, "an array is assigned to a slice");

    Box box = Box{items: [5, 6, 7]};
    check(total(box.items) == 18, "an array field passes as a slice");

    int[2][2] grid = [[1, 2], [3, 4]];
    check(total(grid[1]) == 7, "an array element passes as a slice");

    check(total(nums[1:3]) == 50, "an explicit slice still works");

    check(total([4, 5, 6]) == 15, "an array literal passes as a slice");

    char[3] word = ['a', 'b', 'c'];
    string seen = string(word);
    word[0] = 'x';
    check(seen == "xbc", "a char array converts to a string view of itself");
}
`, expected: []string{"an array variable passes as a slice", "the slice is a view of the array", "an array initialises a slice variable", "an array is assigned to a slice", "an array field passes as a slice", "an array element passes as a slice", "an explicit slice still works", "an array literal passes as a slice", "a char array converts to a string view of itself"}},
	{dir: "main_args", luaOnly: true, src: `func main(string[] args) !int {
    check(len(args) == 0, "main receives the arguments as a string[]");
    for _, a in args {
        emit(a);
    }
    return 0;
}
`, expected: []string{"main receives the arguments as a string[]"}},
	{dir: "cyclic_modules", files: map[string]string{"world.wsp": `import "entity";

public struct World {
    entity.Entity[4] slots;
    int count;
}

public func (*World w) add(int hp) {
    w.slots[w.count] = entity.make(hp);
    w.count++;
}

public func total(*World w) int {
    int sum;
    for i in 0..w.count - 1 {
        sum += entity.health(w.slots[i]);
    }
    return sum;
}

public func isEven(int n) bool {
    if n == 0 {
        return true;
    }
    return entity.isOdd(n - 1);
}
`, "entity.wsp": `import "world";

public struct Entity {
    int hp;
    *world.World owner;
}

public func make(int hp) Entity {
    return Entity{hp: hp};
}

public func health(Entity e) int {
    return e.hp;
}

public func attach(*Entity e, *world.World w) {
    e.owner = w;
}

public func worldCount(Entity e) int {
    return e.owner.count;
}

public func isOdd(int n) bool {
    if n == 0 {
        return false;
    }
    return world.isEven(n - 1);
}
`}, src: `import "world";
import "entity";

func main() {
    world.World w;
    w.add(10);
    w.add(32);
    check(world.total(&w) == 42, "two modules that import each other share their types");

    entity.Entity e = entity.make(5);
    entity.attach(&e, &w);
    check(entity.worldCount(e) == 2, "a struct points at a type from the module that imports it");

    check(world.isEven(10) && !world.isEven(7), "functions call each other across modules");
}
`, expected: []string{"two modules that import each other share their types", "a struct points at a type from the module that imports it", "functions call each other across modules"}},
	{dir: "methods_across_files", files: map[string]string{"shapes/shapes.wsp": `import "shapes/shape";
`, "shapes/shape.wsp": `public struct Shape {
    int w;
    int h;
}
`, "shapes/area.wsp": `import "shapes/shape";

public func (*shape.Shape s) area() int {
    return s.w * s.h;
}

func (*shape.Shape s) double() int {
    return s.area() * 2;
}

public func (*shape.Shape s) twiceArea() int {
    return s.double();
}
`}, src: `import "shapes";

func main() {
    shapes.Shape s = shapes.Shape{w: 3, h: 4};
    check(s.area() == 12, "a method declared in another file of the folder");
    check(s.twiceArea() == 24, "a private method of that file is used by its exported one");
}
`, expected: []string{"a method declared in another file of the folder", "a private method of that file is used by its exported one"}},
	{dir: "until_body_variable", src: `func main() {
    int total;
    loop {
        int n = total + 1;
        total = n * 2;
    } until n > 20;
    check(total == 62, "until reads a variable declared in the body");

    int outside = 0;
    loop {
        outside++;
        if outside == 2 {
            continue;
        }
    } until outside >= 4;
    check(outside == 4, "a continue still reaches the until condition");

    int rounds;
    loop {
        int step = rounds + 1;
        defer rounds = step;
    } until step >= 3;
    check(rounds == 3, "a deferred statement runs before the next round");
}
`, expected: []string{"until reads a variable declared in the body", "a continue still reaches the until condition", "a deferred statement runs before the next round"}},
	{dir: "valueless_handler", src: `enum Fail {
    Ok,
    Bad
}

int trace;

func close(int fd) ! {
    if fd < 0 {
        return Error(Fail.Bad, "bad descriptor");
    }
}

func quiet() {
    defer close(-1) ?? {};
    trace += 1;
}

func loud() {
    defer close(-1) ?? |e| {
        trace += e.line > 0 ? 10 : 0;
    };
    trace += 100;
}

func main() {
    quiet();
    check(trace == 1, "a handler with no value can fall through, even in a defer");

    loud();
    check(trace == 111, "the handler of a deferred call can read the error");

    close(0) ?? {
        trace = 0;
    };
    check(trace == 111, "a call that succeeds skips its handler");
}
`, expected: []string{"a handler with no value can fall through, even in a defer", "the handler of a deferred call can read the error", "a call that succeeds skips its handler"}},
	{dir: "defer_error_bind", src: `enum Fail {
    Ok,
    Boom,
    Bang
}

int trace;

func inner(int n) !int {
    if n == 1 {
        return Error(Fail.Boom, "inner boom");
    }
    return n;
}

func build(int n) !int {
    defer! |e| {
        trace = trace * 10 + e.line;
        if e.message == "inner boom" {
            trace += 1000;
        }
    }
    int v = inner(n)!;
    if v == 2 {
        return Error(Fail.Bang, "bang");
    }
    return v;
}

func main() {
    let _, first = build(1);
    check(first != null && trace >= 1000, "defer! reads an error passed up with !");

    trace = 0;
    let _, second = build(2);
    check(second != null && trace > 0 && trace < 1000, "defer! reads an error returned by return Error");

    trace = 0;
    let value, third = build(3);
    check(third == null && value == 3 && trace == 0, "defer! stays quiet when the function succeeds");
}
`, expected: []string{"defer! reads an error passed up with !", "defer! reads an error returned by return Error", "defer! stays quiet when the function succeeds"}},
	{dir: "empty_literal", src: `struct Holder {
    char[] data;
}

func make() char[] {
    return [];
}

func count(int[] xs) int {
    return len(xs);
}

func main() {
    char[] a = make();
    check(len(a) == 0, "return [] gives an empty slice");

    char[] b = [];
    check(len(b) == 0, "a declaration takes the type of the slice");

    int[] c = [1, 2];
    c = [];
    check(len(c) == 0, "an assignment can empty a slice");

    Holder h = Holder{data: []};
    check(len(h.data) == 0, "a struct field can be []");

    check(count([]) == 0, "an argument can be []");

    int[3] arr = [];
    check(len(arr) == 3 && arr[0] == 0 && arr[2] == 0, "[] is also the zero array");

    free(a, b);
    check(true, "free accepts an empty slice");
}
`, expected: []string{"return [] gives an empty slice", "a declaration takes the type of the slice", "an assignment can empty a slice", "a struct field can be []", "an argument can be []", "[] is also the zero array", "free accepts an empty slice"}},
	{dir: "in_array_literal", src: `func isSpace(char ch) bool {
    return ch in [' ', '\n', '\r', '\t'];
}

func small(int n) bool {
    return n in [1, 2, 3];
}

func main() {
    check(isSpace(' ') && isSpace('\t'), "a char is found in a literal of chars");
    check(!isSpace('x'), "a char that is not in the literal");
    check(small(2) && !small(9), "an int in a literal of ints");
    check(!(7 in [1, 2]), "a literal used directly in the expression");
}
`, expected: []string{"a char is found in a literal of chars", "a char that is not in the literal", "an int in a literal of ints", "a literal used directly in the expression"}},
	{dir: "method_modules", files: map[string]string{"counter.wsp": `public struct Counter {
    int n;
}

public func (*Counter c) bump(int by) {
    c.n += by;
}

public func (*Counter c) get() int {
    return c.n;
}

func (*Counter c) doubled() int {
    return c.n * 2;
}

public func (*Counter c) twice() int {
    return c.doubled();
}

public func start(int n) Counter {
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

	{dir: "type_modules", files: map[string]string{"units.wsp": `public type Meters int;

public enum Level {
    Low,
    High,
}

public func twice(Meters m) Meters {
    return Meters(int(m) * 2);
}

public func isHigh(Level l) bool {
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

	{dir: "ptr_double", src: `struct P {
    int x;
    int y;
}

func setAll(**P pp, int v) {
    pp@.x = v;
    pp@@.y = v + 1;
}

func main() {
    P s = P{x: 1, y: 2};
    *P p = &s;
    **P pp = &p;
    check(pp@.x == 1, "member through a double pointer");
    check(pp@@.y == 2, "member of a fully dereferenced pointer");
    pp@.x = 10;
    check(s.x == 10, "write through a double pointer");
    pp@@.y += 5;
    check(s.y == 7, "compound write of a fully dereferenced pointer");
    setAll(pp, 40);
    check(s.x == 40, "double pointer parameter first field");
    check(s.y == 41, "double pointer parameter second field");
    P other = P{x: 100, y: 200};
    pp@ = &other;
    check(p.x == 100, "writing the inner pointer redirects the pointer");
    check(s.x == 40, "the old target is untouched");
}
`},

	{dir: "ptr_null", src: `struct Node {
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
    n.p@ = 8;
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

	{dir: "method_pointer", src: `struct Counter {
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
        case 1 => 100;
        case 2 => 200;
        default => 0;
    };
}

func main() {
    check(pick(1) == 100, "switch expr first case");
    check(pick(2) == 200, "switch expr second case");
    check(pick(9) == 0, "switch expr default");
    int v = 2;
    let doubled = switch v {
        case 2 => 4;
        default => 0;
    };
    check(doubled == 4, "switch expr let");
    int assigned = 0;
    assigned = switch v {
        case 2 => 7;
        default => 0;
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

	{dir: "err_coalesce_multi", src: `enum Kind {
    A,
    B
}

func h(int x) !(int, int) {
    if x == 0 {
        return Error(Kind.B);
    }
    return x, x * 2;
}

func v(int x) ! {
    if x == 0 {
        return Error(Kind.A);
    }
}

func k(int x) int {
    let a, b = h(x) ?? |e| {
        return -1;
    };
    return a + b;
}

func w(int x) int {
    v(x) ?? |e| {
        return -2;
    };
    return 1;
}

func main() {
    let a, b = h(3) ?? (7, 8);
    check(a == 3 && b == 6, "tuple default is skipped on success");
    let c, d = h(0) ?? (7, 8);
    check(c == 7 && d == 8, "tuple default is used on failure");
    let p, q = h(0) ?? |e| (e.code, 5);
    check(p == 1 && q == 5, "tuple handler sees the error");
    check(k(2) == 6, "block handler with two values on success");
    check(k(0) == -1, "block handler with two values on failure");
    check(w(1) == 1, "block handler without a value on success");
    check(w(0) == -2, "block handler without a value on failure");
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
        case null => v;
        case Kind.A => 100;
        default => 200;
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

	{dir: "abort_index_array", src: `func main() {
    int[5] nums = [1, 2, 3, 4, 5];
    int i = 4;
    check(nums[i] == 5, "abortidx last element is readable");
    i = 5;
    check(true, "abortidx before the bad read");
    int x = nums[i];
    check(false, "abortidx never printed");
}
`, expected: []string{"abortidx last element is readable", "abortidx before the bad read"}, abort: "panic: index out of range"},

	{dir: "abort_index_negative", src: `func main() {
    int[5] nums = [1, 2, 3, 4, 5];
    int i = 0;
    i -= 1;
    check(true, "abortneg before the bad read");
    int x = nums[i];
    check(false, "abortneg never printed");
}
`, expected: []string{"abortneg before the bad read"}, abort: "panic: index out of range"},

	{dir: "abort_index_write", src: `func main() {
    int[3] nums = [1, 2, 3];
    int i = 3;
    check(true, "abortwrite before the bad write");
    nums[i] = 9;
    check(false, "abortwrite never printed");
}
`, expected: []string{"abortwrite before the bad write"}, abort: "panic: index out of range"},

	{dir: "abort_index_span", src: `func main() {
    int[5] nums = [1, 2, 3, 4, 5];
    int[] part = nums[1:3];
    int i = 2;
    check(part[1] == 3, "abortspan last element is readable");
    check(true, "abortspan before the bad read");
    int x = part[i];
    check(false, "abortspan never printed");
}
`, expected: []string{"abortspan last element is readable", "abortspan before the bad read"}, abort: "panic: index out of range"},

	{dir: "abort_null_deref", src: `func main() {
    *int p = null;
    check(true, "abortnull before the dereference");
    int v = p@;
    check(false, "abortnull never printed");
}
`, expected: []string{"abortnull before the dereference"}, abort: "panic: null pointer dereference"},

	{dir: "abort_null_field", src: `struct P {
    int x;
    int y;
}

func main() {
    *P p = null;
    check(true, "abortnullfield before the access");
    int v = p.y;
    check(false, "abortnullfield never printed");
}
`, expected: []string{"abortnullfield before the access"}, abort: "panic: null pointer dereference"},

	{dir: "abort_null_error", src: `enum Kind {
    A
}

func ok() !int {
    return 1;
}

func main() {
    let v, e = ok();
    check(v == 1, "abortnullerr before the access");
    string m = e.message;
    check(false, "abortnullerr never printed");
}
`, expected: []string{"abortnullerr before the access"}, abort: "panic: null pointer dereference"},

	{dir: "abort_slice_end", src: `func main() {
    int[5] nums = [1, 2, 3, 4, 5];
    int end = 6;
    check(true, "abortslice before the bad slice");
    int[] part = nums[1:end];
    check(false, "abortslice never printed");
}
`, expected: []string{"abortslice before the bad slice"}, abort: "panic: slice out of range"},

	{dir: "abort_slice_order", src: `func main() {
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
	{dir: "alloc_basic", src: `func main() ! {
    *int p = malloc(sizeof(int) * 4)!;
    p@ = 42;
    check(p@ == 42, "malloc write and read");
    free(p);
    check(sizeof(int) == 8, "sizeof int");
    check(sizeof(int8) == 1, "sizeof int8");
    check(sizeof(int32) == 4, "sizeof int32");
}
`, expected: []string{"malloc write and read", "sizeof int", "sizeof int8", "sizeof int32"}},

	{dir: "alloc_reuse", src: `func main() ! {
    *int a = malloc(24)!;
    free(a);
    *int b = malloc(24)!;
    check(a == b, "freed block is reused");
    free(b);
    free(null);
    check(true, "free null is a no-op");
}
`, expected: []string{"freed block is reused", "free null is a no-op"}},

	{dir: "alloc_realloc", src: `func main() ! {
    *int p = malloc(8)!;
    p@ = 1234;
    *int same = realloc(p, 8)!;
    check(same == p, "realloc within capacity keeps the block");
    *int big = realloc(p, 100)!;
    check(big@ == 1234, "realloc keeps the contents");
    *int fresh = realloc(null, 50)!;
    check(fresh != null, "realloc of null allocates");
    *int huge = realloc(big, 100000)!;
    check(huge@ == 1234, "realloc to a large block keeps the contents");
    free(huge);
    free(fresh);
}
`, expected: []string{"realloc within capacity keeps the block", "realloc keeps the contents", "realloc of null allocates", "realloc to a large block keeps the contents"}},

	{dir: "abort_malloc_oom", exit: 1, src: `func main() ! {
    check(true, "oom before the failure");
    *int p = malloc(-1)!;
    free(p);
}
`, expected: []string{"oom before the failure"}, abort: "error[runtime.AllocError]: out of memory"},

	{dir: "ptr_index", src: `struct P {
    int a;
    int8 b;
}

func main() ! {
    *int p = malloc(sizeof(int) * 4)!;
    p[0] = 10;
    p[1] = 20;
    p[3] = 40;
    check(p[0] == 10, "pointer index first");
    check(p[1] == 20, "pointer index second");
    check(p[3] == 40, "pointer index last");
    p[1] += 5;
    check(p[1] == 25, "pointer index compound assign");
    *int mid = (*int)(p);
    check(mid[3] == 40, "pointer index through a converted pointer");

    *int8 b = malloc(16)!;
    b[4] = 7;
    *int8 q = (*int8)(b);
    check(q[4] == 7, "pointer index on bytes");
    int i = 4;
    check(b[i] == 7, "pointer index with a variable");

    *P ps = malloc(sizeof(P) * 3)!;
    ps[2].a = 99;
    ps[2].b = 5;
    check(ps[2].a == 99, "pointer index struct field a");
    check(ps[2].b == 5, "pointer index struct field b");
    free(ps);
    free(b);
    free(p);
}
`, expected: []string{"pointer index first", "pointer index second", "pointer index last", "pointer index compound assign", "pointer index through a converted pointer", "pointer index on bytes", "pointer index with a variable", "pointer index struct field a", "pointer index struct field b"}},

	{dir: "strings_core", src: `import "std/strings";

func main() ! {
    char[64] buf;

    check(strings.indexOf("hello", "ll") == 2, "indexOf finds the first match");
    check(strings.indexOf("hello", "") == 0, "indexOf of an empty sub is 0");
    check(strings.indexOf("abc", "c") == 2, "indexOf finds a match at the end");
    check(strings.indexOf("abc", "d") == -1, "indexOf misses");
    check(strings.lastIndexOf("a-b-c", "-") == 3, "lastIndexOf finds the last match");
    check(strings.count("aaaa", "aa") == 2, "count counts separate matches");
    check(!strings.startsWith("ab", "abc"), "startsWith longer than the string is false");
    check(strings.endsWith("hello", "llo"), "endsWith matches the tail");
    check(strings.compare("ab", "abc") == -1 && strings.compare("b", "abc") == 1, "compare orders by prefix length");
    check(strings.equalFold("HeLLo", "hello"), "equalFold ignores ASCII case");

    check(strings.trimSpace("  hi \n") == "hi", "trimSpace");
    check(strings.trim("xxhixx", "x") == "hi", "trim with a charset");
    check(strings.trimPrefix("prefix-name", "prefix-") == "name", "trimPrefix");
    check(strings.trimSuffix("name.wsp", ".wsp") == "name", "trimSuffix");
    let left, right, found = strings.cut("key=value", "=");
    check(found && left == "key" && right == "value", "cut finds the separator");
    let whole, rest, missing = strings.cut("novalue", "=");
    check(!missing && whole == "novalue" && rest == "", "cut without a separator");

    strings.Splitter sp = strings.splitter("a,b,c", ",");
    int pieces;
    loop {
        let piece, ok = sp.next();
        if !ok {
            break;
        }
        pieces++;
    }
    check(pieces == 3, "splitter yields every piece");

    strings.Splitter tail = strings.splitter("a,b,", ",");
    int tailPieces;
    string last = "x";
    loop {
        let piece, ok = tail.next();
        if !ok {
            break;
        }
        tailPieces++;
        last = piece;
    }
    check(tailPieces == 3 && last == "", "splitter keeps the trailing empty piece");

    strings.Splitter none = strings.splitter("", ",");
    let first, firstOk = none.next();
    let second, secondOk = none.next();
    check(firstOk && first == "" && !secondOk, "splitter of an empty string yields one piece");

    strings.Splitter bytes = strings.splitter("abc", "");
    int byteCount;
    loop {
        let piece, ok = bytes.next();
        if !ok {
            break;
        }
        byteCount++;
    }
    check(byteCount == 3, "splitter with an empty separator yields bytes");

    check(strings.concat(buf, "ab", "cd")!! == "abcd", "concat");
    check(strings.concat(buf, "ab", "")!! == "ab", "concat accepts empty parts");
    char[3] small;
    string tooBig = strings.concat(small, "abcd") ?? "full";
    check(tooBig == "full", "concat reports NoSpace");
    check(strings.join(buf, ["a", "b", "c"], ", ")!! == "a, b, c", "join");
    check(strings.rep(buf, "ab", 3)!! == "ababab", "rep");
    check(strings.replace(buf, "a-b-c", "-", "+", -1)!! == "a+b+c", "replace all");
    check(strings.replace(buf, "a-b-c", "-", "<>", -1)!! == "a<>b<>c", "replace with a longer text");
    check(strings.replace(buf, "a--b--c", "--", "-", -1)!! == "a-b-c", "replace with a shorter text");
    check(strings.replace(buf, "a--b--c", "--", "-", 1)!! == "a-b--c", "replace the first only");
    check(strings.replace(buf, "-a-b", "-", "+", -1)!! == "+a+b", "replace with a leading match");
    check(strings.upper(buf, "abC")!! == "ABC", "upper");
    check(strings.lower(buf, "AbC")!! == "abc", "lower");

    char[3] word = ['a', 'b', 'c'];
    strings.toUpper(word);
    check(string(word) == "ABC", "toUpper in place");
    char[4] even = ['a', 'b', 'c', 'd'];
    strings.reverse(even);
    check(string(even) == "dcba", "reverse even length");
    char[5] odd = ['a', 'b', 'c', 'd', 'e'];
    strings.reverse(odd);
    check(string(odd) == "edcba", "reverse odd length");
    char[0] nothing;
    strings.reverse(nothing);
    check(len(nothing) == 0, "reverse empty");

    char[] copy = strings.clone("hey")!;
    check(string(copy) == "hey", "clone copies the bytes");
    free(copy);
    char[] empty = strings.clone("")!;
    check(len(empty) == 0, "clone of an empty string");
}
`, expected: []string{"indexOf finds the first match", "indexOf of an empty sub is 0", "indexOf finds a match at the end", "indexOf misses", "lastIndexOf finds the last match", "count counts separate matches", "startsWith longer than the string is false", "endsWith matches the tail", "compare orders by prefix length", "equalFold ignores ASCII case", "trimSpace", "trim with a charset", "trimPrefix", "trimSuffix", "cut finds the separator", "cut without a separator", "splitter yields every piece", "splitter keeps the trailing empty piece", "splitter of an empty string yields one piece", "splitter with an empty separator yields bytes", "concat", "concat accepts empty parts", "concat reports NoSpace", "join", "rep", "replace all", "replace with a longer text", "replace with a shorter text", "replace the first only", "replace with a leading match", "upper", "lower", "toUpper in place", "reverse even length", "reverse odd length", "reverse empty", "clone copies the bytes", "clone of an empty string"}},

	{dir: "enum_dot_ternary", src: `enum Kind {
    File,
    Dir,
    Other
}

func pick(int m) Kind {
    return switch m {
        case 1 => .File;
        case 2 => .Dir;
        default => .Other;
    };
}

func is(Kind k, Kind want) bool {
    return k == want;
}

func main() {
    int m = 2;
    Kind k = switch m {
        case 1 => .File;
        default => .Dir;
    };
    check(k == Kind.Dir, "a switch expression infers .Member from the declared type");
    check(pick(1) == Kind.File && pick(3) == Kind.Other, "a switch expression infers .Member from the return type");
    Kind t = m == 2 ? .Dir : .File;
    check(t == Kind.Dir, "a ternary infers .Member from the declared type");
    Kind u = Kind.File;
    u = m == 1 ? .File : .Other;
    check(u == Kind.Other, "a ternary infers .Member in an assignment");
    check(is(Kind.Other, m == 2 ? .Other : .File), "a ternary infers .Member as an argument");
    check(pick(2) == (m > 1 ? .Dir : .File), "a ternary infers .Member on the side of a comparison");
}
`, expected: []string{"a switch expression infers .Member from the declared type", "a switch expression infers .Member from the return type", "a ternary infers .Member from the declared type", "a ternary infers .Member in an assignment", "a ternary infers .Member as an argument", "a ternary infers .Member on the side of a comparison"}},

	{dir: "os_getenv", x64Only: true, src: `import "std/os";

func main() {
    let path, found = os.getenv("PATH");
    check(found && len(path) > 0, "getenv finds a variable that is always set");
    let none, missing = os.getenv("WISP_SURELY_UNSET_VARIABLE");
    check(!missing && none == "", "getenv of a missing variable is false");
    let part, partial = os.getenv("PAT");
    check(!partial && part == "", "getenv does not match a prefix of a name");
}
`, expected: []string{"getenv finds a variable that is always set", "getenv of a missing variable is false", "getenv does not match a prefix of a name"}},

	{dir: "path_core", src: `import "std/path";
import "std/strings";

func main() ! {
    char[64] buf;

    check(path.base("a/b") == "b", "base takes the last element");
    check(path.base("a/b/") == "b", "base ignores trailing separators");
    check(path.base("/") == "/" && path.base("///") == "/", "base of only separators is the root");
    check(path.base("") == ".", "base of an empty path is a dot");
    check(path.base("my file ") == "my file ", "base keeps a trailing space");
    check(path.dir("a/b") == "a" && path.dir("a//b") == "a", "dir drops the last element");
    check(path.dir("/a") == "/" && path.dir("//a") == "/", "dir of a path under the root is the root");
    check(path.dir("/") == "/" && path.dir("///") == "/", "dir of the root is the root");
    check(path.dir("a") == "." && path.dir("") == ".", "dir of a single name is a dot");
    check(path.dir("/a/b/") == "/a", "dir ignores trailing separators");
    check(path.dir("../a") == "..", "dir does not clean");
    check(path.ext("a.tar.gz") == ".gz" && path.stem("a.tar.gz") == "a.tar", "ext is the last dot and stem is the rest");
    check(path.ext(".gitignore") == "" && path.stem(".gitignore") == ".gitignore", "a leading dot is part of the name");
    check(path.ext("b.") == "" && path.stem("b.") == "b.", "a trailing dot is part of the name");
    check(path.ext("d/x.y/z") == "", "ext looks only at the base");
    check(path.isAbs("/a") && !path.isAbs("a") && !path.isAbs(""), "isAbs");

    check(path.join(buf, "a", "b")!! == "a/b", "join puts one separator between parts");
    check(path.join(buf, "a//", "//b")!! == "a/b", "join collapses repeated separators");
    check(path.join(buf, "", "a", "", "b")!! == "a/b", "join skips empty parts");
    check(path.join(buf, "/", "a")!! == "/a" && path.join(buf, "/a", "b")!! == "/a/b", "join keeps the root of the first part");
    check(path.join(buf, "/", "/", "a", "b")!! == "/a/b", "join writes the root once");
    check(path.join(buf, "", "/a")!! == "/a", "join takes the root from the first part that is not empty");
    check(path.join(buf, "a", "/")!! == "a", "join of a name and the root is the name");
    check(path.join(buf)!! == "" && path.join(buf, "")!! == "", "join of nothing is empty");
    check(path.join(buf, "/")!! == "/", "join of the root is the root");
    char[3] tiny;
    string cut = path.join(tiny, "abc", "d") ?? "full";
    check(cut == "full", "join reports NoSpace");
    string cutRoot = path.join(tiny, "/ab", "c") ?? "full";
    check(cutRoot == "full", "join with a root reports NoSpace");

    strings.Splitter el = path.elements("/a//b/");
    int count;
    string joined = "";
    loop {
        let piece, ok = el.next();
        if !ok {
            break;
        }
        count++;
        joined = piece;
    }
    check(count == 2 && joined == "b", "elements skips empty pieces");
    strings.Splitter rootEl = path.elements("/");
    let none, rootOk = rootEl.next();
    strings.Splitter emptyEl = path.elements("");
    let nothing, emptyOk = emptyEl.next();
    check(!rootOk && !emptyOk && none == "" && nothing == "", "elements of a root or an empty path is nothing");
    strings.Splitter toks = strings.tokens("a,b,", ",");
    int tokCount;
    loop {
        let piece, ok = toks.next();
        if !ok {
            break;
        }
        tokCount++;
    }
    check(tokCount == 2, "tokens skips the empty piece at the end");

    check(path.clean(buf, "")!! == "." && path.clean(buf, "./")!! == ".", "clean of an empty path is a dot");
    check(path.clean(buf, "a//b")!! == "a/b", "clean collapses separators");
    check(path.clean(buf, "a/./b")!! == "a/b", "clean drops dots");
    check(path.clean(buf, "a/b/../c")!! == "a/c", "clean resolves dot dot");
    check(path.clean(buf, "a/..")!! == ".", "clean can end in a dot");
    check(path.clean(buf, "../../a")!! == "../../a", "clean keeps leading dot dots in a relative path");
    check(path.clean(buf, "a/../..")!! == "..", "clean keeps the dot dot that has nothing to undo");
    check(path.clean(buf, "/..")!! == "/" && path.clean(buf, "/a/../..")!! == "/", "clean never goes above the root");
    check(path.clean(buf, "/a/b/")!! == "/a/b", "clean drops the trailing separator");
    check(path.clean(buf, "a/b/c/../../d")!! == "a/d", "clean undoes several elements");
    string cutClean = path.clean(tiny, "abcd/e") ?? "full";
    check(cutClean == "full", "clean reports NoSpace");
}
`, expected: []string{"base takes the last element", "base ignores trailing separators", "base of only separators is the root", "base of an empty path is a dot", "base keeps a trailing space", "dir drops the last element", "dir of a path under the root is the root", "dir of the root is the root", "dir of a single name is a dot", "dir ignores trailing separators", "dir does not clean", "ext is the last dot and stem is the rest", "a leading dot is part of the name", "a trailing dot is part of the name", "ext looks only at the base", "isAbs", "join puts one separator between parts", "join collapses repeated separators", "join skips empty parts", "join keeps the root of the first part", "join writes the root once", "join takes the root from the first part that is not empty", "join of a name and the root is the name", "join of nothing is empty", "join of the root is the root", "join reports NoSpace", "join with a root reports NoSpace", "elements skips empty pieces", "elements of a root or an empty path is nothing", "tokens skips the empty piece at the end", "clean of an empty path is a dot", "clean collapses separators", "clean drops dots", "clean resolves dot dot", "clean can end in a dot", "clean keeps leading dot dots in a relative path", "clean keeps the dot dot that has nothing to undo", "clean never goes above the root", "clean drops the trailing separator", "clean undoes several elements", "clean reports NoSpace"}},

	{dir: "os_dir", x64Only: true, src: `import "std/os";
import "std/path";
import "std/conv";

func main() ! {
    char[256] buf;
    char[32] num;
    string root = "/tmp/wisp_t_dir";
    os.mkdirAll(path.join(buf, root, "sub")!!)!;
    int fd = os.create(path.join(buf, root, "aa")!!)!;
    os.close(fd);

    os.Dir d = os.openDir(root)!;
    int files, dirs, other;
    bool dots;
    loop {
        let name, kind, ok = d.next()!;
        if !ok {
            break;
        }
        if name == "." || name == ".." {
            dots = true;
        }
        if kind == .File {
            files++;
        } else if kind == .Dir {
            dirs++;
        } else {
            other++;
        }
    }
    d.close();
    check(files == 1 && dirs == 1 && other == 0, "a directory lists its files and subdirectories with their kinds");
    check(!dots, "the entries . and .. are not listed");

    os.Dir empty = os.openDir(path.join(buf, root, "sub")!!)!;
    let nothing, kindNone, more = empty.next()!;
    check(!more && nothing == "", "an empty directory lists nothing");
    empty.close();

    let notDir, problem = os.openDir(path.join(buf, root, "aa")!!);
    check(problem == os.IOError.NotDir, "openDir on a file is NotDir");

    os.mkdirAll(path.join(buf, root, "many")!!)!;
    for i in 1..300 {
        char[256] full;
        int made = os.create(path.join(full, root, "many", conv.itoa(num, i)!!)!!)!;
        os.close(made);
    }
    os.Dir many = os.openDir(path.join(buf, root, "many")!!)!;
    int seen;
    loop {
        let name, kind, ok = many.next()!;
        if !ok {
            break;
        }
        seen++;
    }
    many.close();
    check(seen == 300, "a listing that needs several reads finds every entry");

    for i in 1..300 {
        char[256] full;
        os.remove(path.join(full, root, "many", conv.itoa(num, i)!!)!!)!;
    }
    os.removeDir(path.join(buf, root, "many")!!)!;
    os.remove(path.join(buf, root, "aa")!!)!;
    os.removeDir(path.join(buf, root, "sub")!!)!;
    os.removeDir(root)!;
}
`, expected: []string{"a directory lists its files and subdirectories with their kinds", "the entries . and .. are not listed", "an empty directory lists nothing", "openDir on a file is NotDir", "a listing that needs several reads finds every entry"}},

	{dir: "os_mkdirall", x64Only: true, src: `import "std/os";
import "std/path";

func main() ! {
    char[256] buf;
    string root = "/tmp/wisp_t_mkdirall";
    os.mkdirAll(path.join(buf, root, "a", "b", "c")!!)!;
    check(os.isDir(path.join(buf, root, "a", "b", "c")!!), "mkdirAll creates every missing level");
    os.mkdirAll(path.join(buf, root, "a", "b", "c")!!)!;
    check(os.isDir(path.join(buf, root, "a", "b")!!), "mkdirAll of an existing directory is not an error");
    os.mkdirAll(path.join(buf, root, "a", "b", "d", "e")!!)!;
    check(os.isDir(path.join(buf, root, "a", "b", "d", "e")!!), "mkdirAll creates only what is missing");
    os.mkdirAll(path.join(buf, root, "x", "y")!!)!;
    check(os.isDir(path.join(buf, root, "x", "y")!!), "mkdirAll accepts a trailing separator");

    int fd = os.create(path.join(buf, root, "file")!!)!;
    os.close(fd);
    let onFile = os.mkdirAll(path.join(buf, root, "file")!!);
    check(onFile == os.IOError.NotDir, "mkdirAll over a file is NotDir");
    let underFile = os.mkdirAll(path.join(buf, root, "file", "sub")!!);
    check(underFile == os.IOError.NotDir, "mkdirAll under a file is NotDir");
    let empty = os.mkdirAll("");
    check(empty == os.IOError.NotFound, "mkdirAll of an empty path is NotFound");

    os.removeDir(path.join(buf, root, "a", "b", "d", "e")!!)!;
    os.removeDir(path.join(buf, root, "a", "b", "d")!!)!;
    os.removeDir(path.join(buf, root, "a", "b", "c")!!)!;
    os.removeDir(path.join(buf, root, "a", "b")!!)!;
    os.removeDir(path.join(buf, root, "a")!!)!;
    os.removeDir(path.join(buf, root, "x", "y")!!)!;
    os.removeDir(path.join(buf, root, "x")!!)!;
    os.remove(path.join(buf, root, "file")!!)!;
    os.removeDir(root)!;
}
`, expected: []string{"mkdirAll creates every missing level", "mkdirAll of an existing directory is not an error", "mkdirAll creates only what is missing", "mkdirAll accepts a trailing separator", "mkdirAll over a file is NotDir", "mkdirAll under a file is NotDir", "mkdirAll of an empty path is NotFound"}},

	{dir: "math_core", src: `import "std/math";
import "std/errors";

func main() {
    check(math.min(5) == 5 && math.max(7) == 7, "min and max of a single argument");
    check(math.min(5, 2, 9) == 2 && math.max(5, 2, 9) == 9, "min and max of several arguments");
    check(math.min() == 0 && math.max() == 0, "min and max of nothing are zero");
    check(math.abs(-3) == 3 && math.abs(3) == 3, "abs");
    check(math.clamp(5, 1, 3) == 3 && math.clamp(-5, 1, 3) == 1 && math.clamp(2, 1, 3) == 2, "clamp");
    check(math.mod(-7, 3) == 2 && math.mod(7, -3) == 1 && math.mod(-7, -3) == 2 && math.mod(6, 3) == 0, "mod is never negative");
    check(math.gcd(12, 18) == 6 && math.gcd(4, -6) == 2 && math.gcd(-4, 6) == 2, "gcd is not negative");
    check(math.gcd(0, 5) == 5 && math.gcd(0, 0) == 0, "gcd with a zero");
    check(math.sqrt(0) == 0 && math.sqrt(1) == 1 && math.sqrt(2) == 1 && math.sqrt(15) == 3 && math.sqrt(16) == 4, "sqrt rounds down");
    check(math.sqrt(-9) == 0, "sqrt of a negative is zero");
    check(math.sqrt(math.MAX_INT) == 3037000499 && math.sqrt(9223372030926249001) == 3037000499, "sqrt at the top of the range");
    check(math.popCount(0) == 0 && math.popCount(255) == 8 && math.popCount(0xFFFFFFFFFFFFFFFF) == 64, "popCount");
    check(math.trailingZeros(8) == 3 && math.trailingZeros(1) == 0 && math.trailingZeros(0) == 64 && math.trailingZeros(0x8000000000000000) == 63, "trailingZeros");
    check(math.leadingZeros(1) == 63 && math.leadingZeros(0) == 64 && math.leadingZeros(0x8000000000000000) == 0 && math.leadingZeros(255) == 56, "leadingZeros");
    check(math.MIN_INT == -0x8000000000000000 && math.MAX_INT == 0x7FFFFFFFFFFFFFFF && math.MIN_INT - 1 == math.MAX_INT, "the 64-bit limits wrap around");
    check(math.MIN_INT8 == -128 && math.MAX_INT8 == 127 && math.MIN_INT16 == -32768 && math.MAX_INT16 == 32767 && math.MIN_INT32 == -2147483648 && math.MAX_INT32 == 2147483647, "the signed limits");
    check(math.MAX_UINT8 == 255 && math.MAX_UINT16 == 65535 && math.MAX_UINT32 == 4294967295 && math.MAX_UINT64 == 0xFFFFFFFFFFFFFFFF && math.MAX_UINT == math.MAX_UINT64, "the unsigned limits");
    check(math.checkedAdd(math.MAX_INT, -1)!! == 9223372036854775806, "checkedAdd inside the range");
    check(math.checkedSub(5, 7)!! == -2, "checkedSub inside the range");
    check(math.checkedMul(3037000499, 3037000499)!! == 9223372030926249001, "checkedMul at the top of the range");
    check(math.checkedMul(-3037000499, 3037000499)!! == -9223372030926249001, "checkedMul of mixed signs");
    check(math.checkedMul(-4611686018427387904, 2)!! == math.MIN_INT, "checkedMul that lands on the minimum");
    check(math.checkedMul(0, math.MIN_INT)!! == 0, "checkedMul by zero");
    check(math.checkedAbs(-5)!! == 5, "checkedAbs");
    check(math.pow(2, 62)!! == 4611686018427387904, "pow 2 to the 62");
    check(math.pow(10, 18)!! == 1000000000000000000, "pow 10 to the 18");
    check(math.pow(-2, 63)!! == math.MIN_INT, "pow -2 to the 63 is the minimum");
    check(math.pow(0, 0)!! == 1 && math.pow(5, 0)!! == 1 && math.pow(0, 5)!! == 0 && math.pow(3, 4)!! == 81, "pow with a zero");
    check(math.lcm(4, 6)!! == 12 && math.lcm(-4, 6)!! == 12, "lcm is not negative");
    check(math.lcm(0, 5)!! == 0 && math.lcm(0, 0)!! == 0 && math.lcm(5, 0)!! == 0, "lcm with a zero");
    let ve30, e30 = math.checkedAdd(math.MAX_INT, 1);
    check(e30 == errors.Kind.Overflow, "checkedAdd past the top is Overflow");
    let ve32, e32 = math.checkedAdd(math.MIN_INT, -1);
    check(e32 == errors.Kind.Overflow, "checkedAdd below the bottom is Overflow");
    let ve34, e34 = math.checkedSub(math.MIN_INT, 1);
    check(e34 == errors.Kind.Overflow, "checkedSub below the bottom is Overflow");
    let ve36, e36 = math.checkedSub(0, math.MIN_INT);
    check(e36 == errors.Kind.Overflow, "checkedSub of the minimum from zero is Overflow");
    let ve38, e38 = math.checkedMul(math.MIN_INT, -1);
    check(e38 == errors.Kind.Overflow, "checkedMul of the minimum by -1 is Overflow");
    let ve40, e40 = math.checkedMul(-1, math.MIN_INT);
    check(e40 == errors.Kind.Overflow, "checkedMul of -1 by the minimum is Overflow");
    let ve42, e42 = math.checkedMul(3037000500, 3037000500);
    check(e42 == errors.Kind.Overflow, "checkedMul past the top is Overflow");
    let ve44, e44 = math.checkedMul(4611686018427387904, 2);
    check(e44 == errors.Kind.Overflow, "checkedMul 2^62 by 2 is Overflow");
    let ve46, e46 = math.checkedMul(-4611686018427387904, -2);
    check(e46 == errors.Kind.Overflow, "checkedMul of two negatives past the top is Overflow");
    let ve48, e48 = math.checkedAbs(math.MIN_INT);
    check(e48 == errors.Kind.Overflow, "checkedAbs of the minimum is Overflow");
    let ve50, e50 = math.pow(2, 63);
    check(e50 == errors.Kind.Overflow, "pow past the top is Overflow");
    let ve52, e52 = math.pow(3, 40);
    check(e52 == errors.Kind.Overflow, "pow 3 to the 40 is Overflow");
}
`, expected: []string{"min and max of a single argument", "min and max of several arguments", "min and max of nothing are zero", "abs", "clamp", "mod is never negative", "gcd is not negative", "gcd with a zero", "sqrt rounds down", "sqrt of a negative is zero", "sqrt at the top of the range", "popCount", "trailingZeros", "leadingZeros", "the 64-bit limits wrap around", "the signed limits", "the unsigned limits", "checkedAdd inside the range", "checkedSub inside the range", "checkedMul at the top of the range", "checkedMul of mixed signs", "checkedMul that lands on the minimum", "checkedMul by zero", "checkedAbs", "pow 2 to the 62", "pow 10 to the 18", "pow -2 to the 63 is the minimum", "pow with a zero", "lcm is not negative", "lcm with a zero", "checkedAdd past the top is Overflow", "checkedAdd below the bottom is Overflow", "checkedSub below the bottom is Overflow", "checkedSub of the minimum from zero is Overflow", "checkedMul of the minimum by -1 is Overflow", "checkedMul of -1 by the minimum is Overflow", "checkedMul past the top is Overflow", "checkedMul 2^62 by 2 is Overflow", "checkedMul of two negatives past the top is Overflow", "checkedAbs of the minimum is Overflow", "pow past the top is Overflow", "pow 3 to the 40 is Overflow"}},

	{dir: "switch_tagless", src: `func classify(int a, int b) string {
    return switch {
        case a > b => "greater";
        case a == b => "equal";
        default => "less";
    };
}

func sign(int n) int {
    switch {
        case n > 0:
            return 1;
        case n < 0:
            return -1;
    }
    return 0;
}

func main() {
    check(classify(3, 1) == "greater" && classify(2, 2) == "equal", "a switch without a value picks the first true case");
    check(classify(1, 3) == "less", "a switch without a value falls to default");
    check(sign(5) == 1 && sign(-5) == -1 && sign(0) == 0, "a switch statement without a value");
    int x = 7;
    int y = 0;
    bool either;
    switch {
        case x > 5, y == 1:
            either = true;
        default:
            either = false;
    }
    check(either, "a comma between conditions is an or");
    string size = switch {
        case x < 5 => "small";
        case x < 10 => "medium";
        default => "large";
    };
    check(size == "medium", "a switch without a value initialises a variable");
    let msg = switch {
        case x == 7 && y == 0 => "both";
        default => "other";
    };
    check(msg == "both", "a switch without a value with a compound condition");
}
`, expected: []string{"a switch without a value picks the first true case", "a switch without a value falls to default", "a switch statement without a value", "a comma between conditions is an or", "a switch without a value initialises a variable", "a switch without a value with a compound condition"}},

	{dir: "handler_switch", src: `enum Kind {
    NotFound = 1,
    Denied,
    Other
}

func load(int n) !int {
    switch n {
        case 0:
            return Error(Kind.NotFound);
        case 1:
            return Error(Kind.Denied);
        case 2:
            return Error(Kind.Other);
    }
    return n * 10;
}

func pick(int n) int {
    return load(n) ?? |e| switch e {
        case Kind.NotFound => return -1;
        case Kind.Denied => return -2;
        default => return -3;
    };
}

func sum(int a, int b, int c) int {
    int total;
    for i in 0..2 {
        int n = i == 0 ? a : (i == 1 ? b : c);
        int v = load(n) ?? |e| switch e {
            case Kind.NotFound => continue;
            default => return -100;
        };
        total += v;
    }
    return total;
}

func firstGood(int a, int b, int c) int {
    int total;
    :scan for i in 0..2 {
        int n = i == 0 ? a : (i == 1 ? b : c);
        int v = load(n) ?? |e| switch e {
            case Kind.Denied => break scan;
            default => continue;
        };
        total += v;
    }
    return total;
}

func plain(int n) int {
    return load(n) ?? switch {
        case n == 0 => return 7;
        default => return 8;
    };
}

func main() {
    check(pick(0) == -1 && pick(1) == -2 && pick(2) == -3, "a switch handler picks the result by the error");
    check(sum(5, 0, 6) == 110, "a switch handler can leave the loop with continue");
    check(firstGood(5, 1, 6) == 50, "a switch handler can leave the loop with a labeled break");
    check(plain(0) == 7 && plain(1) == 8, "a switch handler without a value uses plain conditions");
    check(pick(5) == 50, "a switch handler is not used when the call works");
}
`, expected: []string{"a switch handler picks the result by the error", "a switch handler can leave the loop with continue", "a switch handler can leave the loop with a labeled break", "a switch handler without a value uses plain conditions", "a switch handler is not used when the call works"}},

	{dir: "handler_yield", src: `enum Kind {
    NotFound = 1,
    Denied,
    Other
}

int trace;

func load(int n) !int {
    switch n {
        case 0:
            return Error(Kind.NotFound);
        case 1:
            return Error(Kind.Denied);
        case 2:
            return Error(Kind.Other);
    }
    return n * 10;
}

func pair(int n) !(int, int) {
    if n == 0 {
        return Error(Kind.NotFound);
    }
    return n, n * 2;
}

func settle(int n) !int {
    int v = load(n) ?? |e| switch e {
        case Kind.NotFound => 7;
        case Kind.Denied => -2;
        default => return e;
    };
    return v;
}

func main() {
    int a = load(0) ?? |e| {
        yield -1;
    };
    check(a == -1, "yield gives the value of the handler");

    int b = load(0) ?? |e| {
        trace = 5;
        yield e.code * 100;
    };
    check(b == 100 && trace == 5, "yield after other statements keeps their effects");

    let x, y = pair(0) ?? |e| {
        yield 1, 2;
    };
    check(x == 1 && y == 2, "yield with several values");

    let s0 = settle(0)!!;
    let s1 = settle(1)!!;
    check(s0 == 7 && s1 == -2, "a switch handler can yield for one error and leave for another");
    let s2, e2 = settle(2);
    check(e2 == Kind.Other, "a switch handler leaves with the original error");

    int c = load(1) ?? |e| {
        if e == Kind.Denied {
            yield 100;
        }
        yield 200;
    };
    check(c == 100, "yield from one branch of an if");

    int d = load(0) ?? |e| {
        for i in 0..5 {
            if i == 3 {
                yield i * 10;
            }
        }
        yield -1;
    };
    check(d == 30, "yield from inside a loop in the handler");

    trace = 0;
    int f = load(0) ?? |e| {
        defer trace += 1;
        trace += 10;
        yield 5;
    };
    check(f == 5 && trace == 11, "yield runs the defers of the handler block");

    int g = load(4) ?? |e| {
        trace = -1;
        yield 0;
    };
    check(g == 40 && trace == 11, "a handler that does not run leaves the value alone");
}
`, expected: []string{"yield gives the value of the handler", "yield after other statements keeps their effects", "yield with several values", "a switch handler can yield for one error and leave for another", "a switch handler leaves with the original error", "yield from one branch of an if", "yield from inside a loop in the handler", "yield runs the defers of the handler block", "a handler that does not run leaves the value alone"}},

	{dir: "switch_values", src: `enum Kind {
    NotFound = 1,
    Other
}

func load(int n) !int {
    if n == 0 {
        return Error(Kind.NotFound);
    }
    return n * 10;
}

func pair(int n) !(int, int) {
    switch n {
        case 0:
            return Error(Kind.NotFound);
        case 1:
            return Error(Kind.Other);
    }
    return n, n * 2;
}

func twice(int v) int {
    return v * 2;
}

func early(int n) int {
    int v = switch n {
        case 0 => return -1;
        default => n * 2;
    };
    return v + 1;
}

func two(int n) (int, int) {
    return switch n {
        case 1 => 1, 2;
        default => 3, 4;
    };
}

func pairOr(int n) int {
    let a, b = pair(n) ?? |e| switch e {
        case Kind.NotFound => 1, 2;
        default => return -9;
    };
    return a * 100 + b;
}

func sumUntilZero(int a, int b, int c) int {
    int total;
    for i in 0..2 {
        int n = i == 0 ? a : (i == 1 ? b : c);
        total += switch n {
            case 0 => break;
            case 1 => continue;
            default => n;
        };
    }
    return total;
}

func sign(int n) int {
    return switch {
        case n > 0 => 1;
        case n < 0 => return -1;
        default => 0;
    };
}

func main() {
    int x = 1;
    let a, b = switch x {
        case 1 => 10, 20;
        default => 0, 0;
    };
    check(a == 10 && b == 20, "a switch gives two values to a let");

    check(early(0) == -1 && early(3) == 7, "a case can leave the function with return");
    check(sumUntilZero(5, 1, 7) == 12, "a case can continue the loop");
    check(sumUntilZero(5, 0, 7) == 5, "a case can break the loop");
    check(twice(switch x { case 1 => 5; default => 6; }) == 10, "a switch is an argument of a call");
    check(twice(load(0) ?? |e| { yield 5; }) == 10, "a block handler works inside an expression");
    check(pairOr(0) == 102 && pairOr(1) == -9 && pairOr(3) == 306, "a switch handler gives two values");
    let p, q = two(1);
    let r, t = two(2);
    check(p == 1 && q == 2 && r == 3 && t == 4, "a function returns a switch of two values");

    int8 small = 3;
    let w = switch x {
        case 1 => small;
        default => 3;
    };
    int8 same = w;
    check(w == 3 && same == 3, "an untyped arm takes the type of the typed one");
    check(sign(5) == 1 && sign(0) == 0 && sign(-4) == -1, "a switch without a value can leave too");
}
`, expected: []string{"a switch gives two values to a let", "a case can leave the function with return", "a case can continue the loop", "a case can break the loop", "a switch is an argument of a call", "a block handler works inside an expression", "a switch handler gives two values", "a function returns a switch of two values", "an untyped arm takes the type of the typed one", "a switch without a value can leave too"}},

	{dir: "coalesce_short_circuit", src: `enum Kind {
    A = 1
}

int calls;

func bump() !int {
    calls++;
    return 1;
}

func fail() !int {
    calls++;
    return Error(Kind.A);
}

func count() int {
    calls++;
    return 1;
}

func main() {
    bool r = false && (bump() ?? 0) == 1;
    check(calls == 0 && !r, "a ?? in an untaken && branch does not run");
    bool s = true || (bump() ?? 0) == 1;
    check(calls == 0 && s, "a ?? in an untaken || branch does not run");
    int t = true ? 5 : (bump() ?? 0);
    int u = false ? (bump() ?? 0) : 6;
    check(calls == 0 && t == 5 && u == 6, "a ?? in an untaken ternary branch does not run");

    bool v = true && (fail() ?? |e| {
        yield 3;
    }) == 3;
    int w = true ? (fail() ?? 4) : 0;
    check(calls == 2 && v && w == 4, "a block handler in a taken branch runs once");

    int x = false && true ? 0 : 1;
    bool y = false && (switch x {
        case 1 => count();
        default => 0;
    }) == 1;
    check(calls == 2 && !y, "a switch in an untaken branch does not run");
}
`, expected: []string{"a ?? in an untaken && branch does not run", "a ?? in an untaken || branch does not run", "a ?? in an untaken ternary branch does not run", "a block handler in a taken branch runs once", "a switch in an untaken branch does not run"}},

	{dir: "cycles_calls", files: map[string]string{"even.wsp": `import "odd";

public const LIMIT = 100;

public func isEven(int n) bool {
    if n == 0 {
        return true;
    }
    return odd.isOdd(n - 1);
}

public func limit() int {
    return odd.LIMIT * 2;
}
`, "odd.wsp": `import "even";

public const LIMIT = 50;

public func isOdd(int n) bool {
    if n == 0 {
        return false;
    }
    return even.isEven(n - 1);
}

public func cap() int {
    return even.LIMIT + 1;
}
`}, src: `import "even";
import "odd";

func main() {
    check(even.isEven(10) && !even.isEven(7) && odd.isOdd(7), "two modules call each other");
    check(even.limit() == 100, "a function reaches the other module through the cycle");
    check(odd.cap() == 101, "a constant of the other module is read after loading");
}
`, expected: []string{"two modules call each other", "a function reaches the other module through the cycle", "a constant of the other module is read after loading"}},

	{dir: "lua_keywords", src: `import "std/math";

struct Box {
    int end;
    int function;
    int local;
}

const then = 5;
int goto = 6;

func repeat(int not, int or) int {
    return not + or;
}

func nil(int and) int => and * 2;

func risky(int n) !int {
    if n == 0 {
        return Error(Kind.A);
    }
    return n;
}

enum Kind {
    A = 1
}

func main() {
    int end = 1;
    int do = 2;
    int elseif = 3;
    int while = 4;
    int total = repeat(end, do) + nil(elseif) + while + then + goto;
    check(total == 24, "Lua keywords work as names of variables, parameters and functions");

    Box box = Box{end: 7, function: 8, local: 9};
    box.end = box.end + 1;
    check(box.end == 8 && box.function == 8 && box.local == 9, "Lua keywords work as names of fields");

    for local in 0..2 {
        total += local;
    }
    int v = risky(0) ?? |or| {
        yield 100;
    };
    check(total == 27 && v == 100, "Lua keywords work as loop and handler names");

    check(math.max(total, 1) == 27 && math.MIN_INT < 0, "a module named math does not hide the Lua math library");
}
`, expected: []string{"Lua keywords work as names of variables, parameters and functions", "Lua keywords work as names of fields", "Lua keywords work as loop and handler names", "a module named math does not hide the Lua math library"}},

	{dir: "target_files", files: map[string]string{"where/where.wsp": `public func name() int {
    return which() + extra();
}

public func pieces() int {
    return both();
}
`, "where/where_x64.wsp": `func which() int {
    return 10;
}

func both() int {
    return 100;
}
`, "where/where_linux_x64.wsp": `func extra() int {
    return 1;
}
`, "where/where_lua.wsp": `func which() int {
    return 10;
}

func extra() int {
    return 1;
}

func both() int {
    return 100;
}
`}, src: `import "where";

func main() {
    check(where.name() == 11, "a file named after a target is part of the module on that target only");
    check(where.pieces() == 100, "a file named after an OS and arch is part of the module on that platform only");
}
`, expected: []string{"a file named after a target is part of the module on that target only", "a file named after an OS and arch is part of the module on that platform only"}},

	{dir: "ptr_slice", src: `func main() ! {
    *int8 b = malloc(16)!;
    for i in 0..15 {
        b[i] = int8(i * 2);
    }
    int8[] s = b[2:6];
    check(len(s) == 4, "pointer slice length");
    check(s[0] == 4, "pointer slice first");
    check(s[3] == 10, "pointer slice last");
    s[1] = 99;
    check(b[3] == 99, "pointer slice writes through to the buffer");
    int8[] t = s[1:3];
    check(t[0] == 99, "slice of a pointer slice");
    int lo = 4;
    int hi = 9;
    int8[] u = b[lo:hi];
    check(len(u) == 5, "pointer slice with variable bounds");
    int8[] head = b[:3];
    check(len(head) == 3 && head[2] == 4, "pointer slice with the start omitted begins at the pointer");
    free(b);
}
`, expected: []string{"pointer slice length", "pointer slice first", "pointer slice last", "pointer slice writes through to the buffer", "slice of a pointer slice", "pointer slice with variable bounds", "pointer slice with the start omitted begins at the pointer"}},

	{dir: "ptr_convert", x64Only: true, src: `struct P {
    int a;
    int b;
}

func main() ! {
    *int p = malloc(32)!;
    p[0] = 1234;
    uint64 addr = uint64(p);
    check(addr != 0, "pointer to integer is not null");
    int signed = int(p);
    check(uint64(signed) == addr, "pointer to int matches uint64");
    *int q = (*int)(addr);
    check(q@ == 1234, "integer to pointer round trip");
    (*int)(addr)@ = 77;
    check(p[0] == 77, "write through a converted integer");
    int v = (*int)(addr)@;
    check(v == 77, "read through a converted integer");

    *P sp = (*P)(p);
    sp.b = 5;
    check(p[1] == 5, "struct pointer reinterpreted from int pointer");
    *int8 bytes = (*int8)(sp);
    check(bytes[0] == 77, "bytes view of a struct pointer");
    check((*int)(0) == null, "converted zero is the null pointer");
    free(p);
}
`, expected: []string{"pointer to integer is not null", "pointer to int matches uint64", "integer to pointer round trip", "write through a converted integer", "read through a converted integer", "struct pointer reinterpreted from int pointer", "bytes view of a struct pointer", "converted zero is the null pointer"}},

	{dir: "ptr_arith", src: `struct P {
    int a;
    int8 b;
}

func main() ! {
    *int p = malloc(sizeof(int) * 8)!;
    *int q = p + 2;
    q@ = 7;
    check(p[2] == 7, "pointer plus offset");
    check(q - p == 2, "pointer difference");
    check(2 + p == q, "offset plus pointer");
    check(q > p, "pointer ordering");
    check(p < q, "pointer ordering reversed");
    q -= 1;
    check(q - p == 1, "pointer minus assign");
    q += 3;
    check(q - p == 4, "pointer plus assign");
    check(q - 4 == p, "pointer minus offset");

    *int8 b = malloc(16)!;
    *int8 c = b + 5;
    check(c - b == 5, "byte pointer difference");
    check(uint64(c) - uint64(b) == 5, "byte pointer address step");

    *P ps = malloc(sizeof(P) * 4)!;
    *P p2 = ps + 1;
    check(p2 - ps == 1, "struct pointer difference");
    check(uint64(p2) - uint64(ps) == uint64(sizeof(P)), "struct pointer steps by sizeof");

    int n = 3;
    check((p + n) - p == 3, "pointer plus variable");
    int32 small = 2;
    check((p + small) - p == 2, "pointer plus narrow integer");

    *int w = p;
    w++;
    check(w - p == 1, "pointer increment steps one element");
    w++;
    w--;
    check(w - p == 1, "pointer decrement steps one element");

    *int8 bw = b;
    bw++;
    check(bw - b == 1, "byte pointer increment");

    *P sw = ps;
    sw++;
    check(uint64(sw) - uint64(ps) == uint64(sizeof(P)), "struct pointer increment");
    free(ps);
    free(b);
    free(p);
}
`, expected: []string{"pointer plus offset", "pointer difference", "offset plus pointer", "pointer ordering", "pointer ordering reversed", "pointer minus assign", "pointer plus assign", "pointer minus offset", "byte pointer difference", "byte pointer address step", "struct pointer difference", "struct pointer steps by sizeof", "pointer plus variable", "pointer plus narrow integer", "pointer increment steps one element", "pointer decrement steps one element", "byte pointer increment", "struct pointer increment"}},

	{dir: "string_from_bytes", x64Only: true, src: `func main() ! {
    *int8 b = malloc(8)!;
    b[0] = 104;
    b[1] = 105;
    string s = string(b[0:2]);
    check(len(s) == 2, "string from int8 pointer slice length");
    check(s == "hi", "string from int8 pointer slice content");
    b[1] = 111;
    check(s == "ho", "string is a view of the buffer");

    *char c = malloc(4)!;
    c[0] = 'o';
    c[1] = 'k';
    char[] cs = c[0:2];
    check(string(cs) == "ok", "string from char slice");

    uint8[4] arr;
    arr[0] = 65;
    arr[1] = 66;
    arr[2] = 67;
    uint8[] us = arr[0:3];
    check(string(us) == "ABC", "string from uint8 array slice");
    check(string(arr[1:3]) == "BC", "string from inline slice");
    check(len(string(us)) == 3, "string from bytes length");
    free(b);
    free(c);
}
`, expected: []string{"string from int8 pointer slice length", "string from int8 pointer slice content", "string is a view of the buffer", "string from char slice", "string from uint8 array slice", "string from inline slice", "string from bytes length"}},

	{dir: "free_slice", src: `func main() ! {
    *int8 b = malloc(24)!;
    b[0] = 120;
    int8[] s = b[0:5];
    free(s);
    *int8 c = malloc(24)!;
    check(b == c, "free of a slice releases its block");
    free(c);
    free(null);
    check(true, "free of null still works");
}
`, expected: []string{"free of a slice releases its block", "free of null still works"}},

	{dir: "free_many", x64Only: true, src: `func main() ! {
    *int a = malloc(24)!;
    *int b = malloc(24)!;
    *int c = malloc(100)!;
    free(a, b, c);
    *int x = malloc(24)!;
    *int y = malloc(24)!;
    *int z = malloc(100)!;
    check((x == a && y == b) || (x == b && y == a), "free of many releases every small block");
    check(z == c, "free of many releases the last block");

    *int8 raw = malloc(40)!;
    int8[] s = raw[0:3];
    *int d = malloc(40)!;
    free(s, d);
    *int8 raw2 = malloc(40)!;
    *int d2 = malloc(40)!;
    check((raw2 == raw && uint64(d2) == uint64(d)) || (uint64(raw2) == uint64(d) && d2 == (*int)(raw)), "free of mixed slice and pointer");
    free(raw2, d2, null);
    check(true, "free of many accepts null");
}
`, expected: []string{"free of many releases every small block", "free of many releases the last block", "free of mixed slice and pointer", "free of many accepts null"}},

	{dir: "free_node_next", src: `struct Node {
    *Node next;
    int v;
}

func main() ! {
    *Node first = malloc(sizeof(Node))!;
    *Node second = malloc(sizeof(Node))!;
    first.next = second;
    second.v = 42;
    second.next = null;
    free(first, first.next);
    *Node a = malloc(sizeof(Node))!;
    *Node b = malloc(sizeof(Node))!;
    check((a == first && b == second) || (a == second && b == first), "free reads every argument before releasing any");
}
`, expected: []string{"free reads every argument before releasing any"}},

	{dir: "string_order", src: `func main() {
    check("a" < "b", "string less");
    check("b" > "a", "string greater");
    check("abc" < "abd", "string less than prefix extension");
    check("ab" < "abc", "string prefix is smaller");
    check("abc" > "ab", "string longer is greater");
    check("" < "a", "empty is smaller");
    checkEq("a" < "a", false, "equal is not less");
    checkEq("a" > "a", false, "equal is not greater");
    check("a" <= "a", "equal is less or equal");
    check("a" >= "a", "equal is greater or equal");
    check("a" <= "b", "less is less or equal");
    check("b" >= "a", "greater is greater or equal");
    checkEq("b" <= "a", false, "greater is not less or equal");
    checkEq("a" >= "b", false, "less is not greater or equal");
    check("abc" > "abb", "last byte decides");
    check("b" > "abc", "first byte decides over length");
    string word = "xxcdxx";
    check(word[2:4] < "ce", "ordering on a slice");
    string left = "apple";
    string right = "apricot";
    check(left < right, "ordering with variables");
    check("" >= "", "empty equals empty");
}
`, expected: []string{"string less", "string greater", "string less than prefix extension", "string prefix is smaller", "string longer is greater", "empty is smaller", "equal is not less", "equal is not greater", "equal is less or equal", "equal is greater or equal", "less is less or equal", "greater is greater or equal", "greater is not less or equal", "less is not greater or equal", "last byte decides", "first byte decides over length", "ordering on a slice", "ordering with variables", "empty equals empty"}},

	{dir: "global_basic", src: `const Base = 10;

enum Mode {
    Off,
    On
}

struct P {
    int x;
    int y;
}

int counter = 0;
int8 small;
int sized = Base * 2 + 1;
string name = "wisp";
let inferred = 7;
Mode mode = Mode.On;
int[3] table = [1, 2, 3];
P origin = P{x: 1, y: 2};
int first, second = 4, 5;
bool flag = true;
char letter = 'q';

func above() int {
    return later;
}

int later = 99;

func bump() {
    counter += 1;
    counter++;
}

func main() {
    check(counter == 0 && small == 0, "global zero initializer");
    check(sized == 21, "global constant expression initializer");
    check(name == "wisp", "global string initializer");
    check(inferred == 7, "global inferred initializer");
    check(mode == Mode.On, "global enum initializer");
    check(table[0] == 1 && table[2] == 3, "global array initializer");
    check(origin.x == 1 && origin.y == 2, "global struct initializer");
    check(first == 4 && second == 5, "global multiple declaration");
    check(flag && letter == 'q', "global bool and char");
    counter += 5;
    check(counter == 5, "global compound assign");
    counter++;
    check(counter == 6, "global increment");
    table[1] = 20;
    check(table[1] == 20, "global array element write");
    origin.x = 9;
    check(origin.x == 9 && origin.y == 2, "global struct field write");
    name = "other";
    check(name == "other", "global string write");
    check(above() == 99, "global used by a function above it");
    flag = false;
    check(!flag, "global bool write");
    bump();
    bump();
    check(counter == 10, "global is shared across calls");
}
`, expected: []string{"global zero initializer", "global constant expression initializer", "global string initializer", "global inferred initializer", "global enum initializer", "global array initializer", "global struct initializer", "global multiple declaration", "global bool and char", "global compound assign", "global increment", "global array element write", "global struct field write", "global string write", "global used by a function above it", "global bool write", "global is shared across calls"}},

	{dir: "fmt_core", x64Only: true, src: `import "std/fmt";
import "std/os";
import "std/errors";

enum Color {
    Red,
    Green,
}

char[256] scratch;

func fm(string format, ...args) string {
    let n, err = fmt.sprintf(scratch[:], format, args...);
    if err != null {
        return "<error>";
    }
    return string(scratch[0:n]);
}

func eq(string got, string want, string what) {
    check(got == want, what);
}

type Meters int;

func fl(string format, ...args) string {
    fmt.Writer w = fmt.Writer{fd: -1, buf: scratch[:], mode: fmt.BufMode.Full};
    let _, _ = w.format(format, args...);
    return string(scratch[0:w.buffered()]);
}

func fe(string format, ...args) Error {
    fmt.Writer w = fmt.Writer{fd: -1, buf: scratch[:], mode: fmt.BufMode.Full};
    let _, err = w.format(format, args...);
    return err;
}

func main() {
    uint big = 18446744073709551615;
    int8 small = -5;
    eq(fm("{}", 42), "42", "decimal");
    eq(fm("{}", -42), "-42", "negative decimal");
    eq(fm("{}", 0), "0", "zero");
    eq(fm("{}", big), "18446744073709551615", "largest uint");
    eq(fm("{}", small), "-5", "int8");
    eq(fm("{d}", 7), "7", "d type letter");
    eq(fm("{x}", 255), "ff", "hex lower case");
    eq(fm("{X}", 255), "FF", "hex upper case");
    eq(fm("{#x}", 255), "0xff", "hex prefix");
    eq(fm("{b}", 5), "101", "binary");
    eq(fm("{#b}", 5), "0b101", "binary prefix");
    eq(fm("{o}", 8), "10", "octal");
    eq(fm("{#o}", 8), "0o10", "octal prefix");
    eq(fm("{x}", -255), "-ff", "negative hex");
    eq(fm("{5}", 42), "   42", "number aligns right");
    eq(fm("{<5}", 42), "42   ", "left align");
    eq(fm("{^6}", 42), "  42  ", "center with even padding");
    eq(fm("{^5}", 42), " 42  ", "center with odd padding puts the extra space on the right");
    eq(fm("{05}", 42), "00042", "zero padding");
    eq(fm("{05}", -42), "-0042", "zero padding goes after the sign");
    eq(fm("{05x}", 255), "000ff", "zero padded hex");
    eq(fm("{#06x}", 255), "0x00ff", "zeros go after the prefix");
    eq(fm("{1}", 1234), "1234", "width smaller than the text changes nothing");
    eq(fm("{-<6}", 7), "7-----", "custom fill");
    eq(fm("{*^7}", "ab"), "**ab***", "custom fill with center");
    eq(fm("{}", "hi"), "hi", "string");
    eq(fm("{5}", "ab"), "ab   ", "string aligns left");
    eq(fm("{>5}", "ab"), "   ab", "string right align");
    eq(fm("{.3}", "abcdef"), "abc", "precision cuts a string");
    eq(fm("{5.3}", "abcdef"), "abc  ", "width and precision");
    eq(fm("[{}]", ""), "[]", "empty string");
    eq(fm("{}", 'x'), "x", "char");
    eq(fm("{d}", 'A'), "65", "char as a number");
    eq(fm("{x}", 'A'), "41", "char in hex");
    eq(fm("{c}", 65), "A", "integer as a char");
    eq(fm("{}", true), "true", "true");
    eq(fm("{}", false), "false", "false");
    eq(fm("{6}", true), "true  ", "bool aligns left");
    eq(fm("{}", Color.Green), "1", "an enum prints its number");
    eq(fm("{}", Color.Green.name()), "Green", "an enum by name through name()");
    eq(fm("plain"), "plain", "plain text");
    eq(fm("a{}b", 1), "a1b", "text around a placeholder");
    eq(fm("{} {} {}", 1, "two", true), "1 two true", "several placeholders of different types");
    eq(fm("{{}}"), "{}", "escaped braces");
    eq(fm("{{{}", 5), "{5", "an escaped brace before a placeholder");
    eq(fm("{}}}", 5), "5}", "an escaped brace after a placeholder");
    eq(fm("a {} b {x}", 1), "a 1 b {x}", "fewer arguments keep the placeholder as written");
    eq(fm("{}"), "{}", "no arguments");
    Meters meters = Meters(3);
    check(fl("a {x} b", "hi") == "a {x} b" && fe("{x}", "hi") == fmt.FormatError.WrongType, "a type letter that does not fit the type is written back and reported as WrongType");
    check(fl("{5.}", 1) == "{5.}" && fe("{5.}", 1) == fmt.FormatError.BadSpec, "a number with no digits in a spec is BadSpec");
    check(fl("{z}", 1) == "{z}" && fe("{z}", 1) == fmt.FormatError.BadSpec, "an unknown type letter is BadSpec");
    check(fl("{x5}", 1) == "{x5}" && fe("{x5}", 1) == fmt.FormatError.BadSpec, "text after the type letter is BadSpec");
    check(fl("a { b", 1) == "a { b" && fe("a { b", 1) == fmt.FormatError.Unmatched, "an open brace with no close is Unmatched");
    check(fl("a } b") == "a } b" && fe("a } b") == fmt.FormatError.Unmatched, "a lone close brace is Unmatched");
    check(fl("a", 1) == "a" && fe("a", 1) == fmt.FormatError.ExtraArgument, "an extra argument is ExtraArgument");
    check(fl("{}", meters) == "{}" && fe("{}", meters) == fmt.FormatError.UnsupportedType, "an argument of a named type is UnsupportedType");
    check(fl("{x} and {}", "hi", 7) == "{x} and 7", "a problem does not stop the text after it, and takes its argument");
    check(fe("{} {}", 1) == null && fl("{} {}", 1) == "1 {}", "fewer arguments are not an error");
    check(fe("{} {x}", 1, 255) == null, "a good format reports no error");
    int x = 5;
    *int p = &x;
    string shown = fm("{}", p);
    check(len(shown) > 2 && shown[0:2] == "0x", "a pointer starts with 0x");
    check(shown == fm("{p}", p), "p is the default for a pointer");

    char[16] out;
    let n, e1 = fmt.sprintf(out[:], "ab{}", 12);
    check(e1 == null && n == 4, "sprintf returns the byte count");
    char[3] tiny;
    let _, e2 = fmt.sprintf(tiny[:], "abcdef");
    check(e2 == errors.Kind.NoSpace, "sprintf into a buffer that is too small is NoSpace");

    char[32] mem;
    fmt.Writer w = fmt.Writer{fd: -1, buf: mem[:], mode: fmt.BufMode.Full};
    let k, _ = w.format("x={}", 5);
    check(k == 3 && w.buffered() == 3 && string(mem[0:3]) == "x=5", "a memory writer collects the text");
    let _, _ = w.format("y");
    check(w.buffered() == 4 && string(mem[0:4]) == "x=5y", "a second call appends");
    char[4] narrow;
    fmt.Writer ws = fmt.Writer{fd: -1, buf: narrow[:], mode: fmt.BufMode.Full};
    let _, e3 = ws.format("abcde");
    check(e3 == errors.Kind.NoSpace, "a memory writer that fills up is NoSpace");

    let file, _ = os.create("fmt_core.txt");
    char[64] store;
    fmt.Writer full = fmt.Writer{fd: file, buf: store[:], mode: fmt.BufMode.Full};
    let _, _ = full.format("hello");
    let held, _ = os.size(file);
    check(held == 0, "a full writer holds the text until flush");
    let _ = full.flush();
    let flushed, _ = os.size(file);
    check(flushed == 5, "flush writes the text");

    let lineFile, _ = os.create("fmt_line.txt");
    char[64] lineStore;
    fmt.Writer line = fmt.Writer{fd: lineFile, buf: lineStore[:], mode: fmt.BufMode.Line};
    let _, _ = line.format("ab");
    let beforeNewline, _ = os.size(lineFile);
    check(beforeNewline == 0, "a line writer holds the text until a newline");
    let _, _ = line.format("c\n");
    let afterNewline, _ = os.size(lineFile);
    check(afterNewline == 4, "a line writer flushes at the newline");

    let rawFile, _ = os.create("fmt_none.txt");
    char[8] rawStore;
    fmt.Writer raw = fmt.Writer{fd: rawFile, buf: rawStore[:], mode: fmt.BufMode.None};
    let _, _ = raw.format("xyz");
    let atOnce, _ = os.size(rawFile);
    check(atOnce == 3, "a writer with no buffering writes at once");

    let streamFile, _ = os.create("fmt_stream.txt");
    char[4] streamStore;
    fmt.Writer stream = fmt.Writer{fd: streamFile, buf: streamStore[:], mode: fmt.BufMode.Full};
    let _, _ = stream.format("abcdefghij");
    let _ = stream.flush();
    let streamed, _ = os.size(streamFile);
    check(streamed == 10, "a small buffer is flushed as it fills");

    let again, _ = os.create("fmt_call.txt");
    let _, _ = fmt.fprintf(again, "n={}", 42);
    let wrote, _ = os.size(again);
    check(wrote == 4, "fprintf writes everything in one call");

    let badFile, _ = os.create("fmt_bad.txt");
    let _, problem = fmt.fprintf(badFile, "a{b", 1);
    let written, _ = os.size(badFile);
    check(problem == fmt.FormatError.Unmatched && written == 3, "fprintf still writes the text when the format has a problem");
    let _ = os.remove("fmt_bad.txt");

    let _ = os.remove("fmt_core.txt");
    let _ = os.remove("fmt_line.txt");
    let _ = os.remove("fmt_none.txt");
    let _ = os.remove("fmt_stream.txt");
    let _ = os.remove("fmt_call.txt");
}
`, expected: []string{"decimal", "negative decimal", "zero", "largest uint", "int8", "d type letter", "hex lower case", "hex upper case", "hex prefix", "binary", "binary prefix", "octal", "octal prefix", "negative hex", "number aligns right", "left align", "center with even padding", "center with odd padding puts the extra space on the right", "zero padding", "zero padding goes after the sign", "zero padded hex", "zeros go after the prefix", "width smaller than the text changes nothing", "custom fill", "custom fill with center", "string", "string aligns left", "string right align", "precision cuts a string", "width and precision", "empty string", "char", "char as a number", "char in hex", "integer as a char", "true", "false", "bool aligns left", "an enum prints its number", "an enum by name through name()", "plain text", "text around a placeholder", "several placeholders of different types", "escaped braces", "an escaped brace before a placeholder", "an escaped brace after a placeholder", "fewer arguments keep the placeholder as written", "no arguments", "a type letter that does not fit the type is written back and reported as WrongType", "a number with no digits in a spec is BadSpec", "an unknown type letter is BadSpec", "text after the type letter is BadSpec", "an open brace with no close is Unmatched", "a lone close brace is Unmatched", "an extra argument is ExtraArgument", "an argument of a named type is UnsupportedType", "a problem does not stop the text after it, and takes its argument", "fewer arguments are not an error", "a good format reports no error", "a pointer starts with 0x", "p is the default for a pointer", "sprintf returns the byte count", "sprintf into a buffer that is too small is NoSpace", "a memory writer collects the text", "a second call appends", "a memory writer that fills up is NoSpace", "a full writer holds the text until flush", "flush writes the text", "a line writer holds the text until a newline", "a line writer flushes at the newline", "a writer with no buffering writes at once", "a small buffer is flushed as it fills", "fprintf writes everything in one call", "fprintf still writes the text when the format has a problem"}},

	{dir: "variadic_method", src: `struct Counter {
    int seen;
}

func (*Counter c) count(...args) int {
    c.seen += len(args);
    return len(args);
}

func (*Counter c) sum(string label, ...args) int {
    int total = 0;
    for _, a in args {
        switch let v = a {
            case int:
                total += v;
            case string:
                total += len(v);
        }
    }
    return total + len(label);
}

func (*Counter c) forward(...args) int {
    return c.count(args...);
}

func (*Counter c) prefixed(...args) int {
    return c.sum("ab", args...);
}

func main() {
    Counter c = Counter{seen: 0};
    check(c.count() == 0, "a variadic method with no extra arguments");
    check(c.count(7) == 1, "one extra argument");
    check(c.count(1, "x", true) == 3, "several arguments of different types");
    check(c.seen == 4, "the receiver is the one the call was made on");
    check(c.sum("abc", 5, "xy") == 10, "a fixed argument and the extras are both read");
    check(c.forward(1, 2) == 2, "a method forwards its variadic to another method");
    check(c.prefixed(10) == 12, "explicit arguments before a spread in a method call");
    check(c.count() == 0 && c.seen == 6, "an empty variadic after other calls");
}
`, expected: []string{"a variadic method with no extra arguments", "one extra argument", "several arguments of different types", "the receiver is the one the call was made on", "a fixed argument and the extras are both read", "a method forwards its variadic to another method", "explicit arguments before a spread in a method call", "an empty variadic after other calls"}},

	{dir: "handler_no_bind", src: `enum Fail {
    Ok,
    Bad
}

func risky(int n) !int {
    if n < 0 {
        return Error(Fail.Bad);
    }
    return n * 2;
}

func firstOr(int n) int {
    int v = risky(n) ?? {
        return -1;
    };
    return v;
}

func sum(int a, int b) int {
    int total = 0;
    for i in 0..1 {
        int x = risky(i == 0 ? a : b) ?? {
            continue;
        };
        total += x;
    }
    return total;
}

func firstGood(int a, int b, int c) int {
    int found = -1;
    for i in 0..2 {
        int x = risky(i == 0 ? a : (i == 1 ? b : c)) ?? continue;
        found = x;
        break;
    }
    return found;
}

func stopAtBad(int a, int b, int c) int {
    int total = 0;
    for i in 0..2 {
        int x = risky(i == 0 ? a : (i == 1 ? b : c)) ?? break;
        total += x;
    }
    return total;
}

func namedJump(int a, int b) int {
    int total = 0;
    for i in 0..1 {
        int x = risky(i == 0 ? a : b) ?? |_| continue;
        total += x;
    }
    return total;
}

int counted = 0;

func bump(int n) {
    risky(n) ?? return;
    counted += 1;
}

func inline(int n) int {
    int a = risky(n) ?? return -2;
    return a;
}

func named(int n) int {
    int a = risky(n) ?? |_| return -3;
    return a;
}

func main() {
    bump(1);
    bump(-1);
    bump(2);
    check(counted == 2, "return with no value after ?? leaves a function that returns nothing");
    check(firstGood(-1, -1, 4) == 8 && firstGood(-1, 3, 4) == 6, "continue after ?? skips the failures");
    check(stopAtBad(1, 2, 3) == 12 && stopAtBad(1, -1, 3) == 2, "break after ?? stops at the first failure");
    check(namedJump(2, -1) == 4 && namedJump(-1, 5) == 10, "continue after ?? with a name that is not used");
    check(inline(5) == 10 && inline(-1) == -2, "return after ?? with no name");
    check(named(5) == 10 && named(-1) == -3, "return after ?? with a name that is not used");
    check(firstOr(4) == 8, "a handler without a name is skipped on success");
    check(firstOr(-1) == -1, "a handler without a name runs on failure");
    check(sum(1, 2) == 6, "continue works in a handler without a name");
    check(sum(-1, 2) == 4, "a failing call is skipped by continue");
}
`, expected: []string{"return with no value after ?? leaves a function that returns nothing", "continue after ?? skips the failures", "break after ?? stops at the first failure", "continue after ?? with a name that is not used", "return after ?? with no name", "return after ?? with a name that is not used", "a handler without a name is skipped on success", "a handler without a name runs on failure", "continue works in a handler without a name", "a failing call is skipped by continue"}},

	{dir: "assert_unwrap", src: `enum Fail {
    Ok,
    Bad
}

func risky(int n) !int {
    if n < 0 {
        return Error(Fail.Bad, "negative input");
    }
    return n * 2;
}

func work(int n) ! {
    if n < 0 {
        return Error(Fail.Bad, "bad work");
    }
}

func main() {
    int a = risky(3)!!;
    check(a == 6, "!! gives the value of a call that works");
    check(risky(4)!! + risky(5)!! == 18, "!! works inside an expression, in a function that cannot fail");
    work(1)!!;
    check(true, "!! works on a call with no value");
    int b = risky(-1)!!;
    check(false, "not reached");
}
`, expected: []string{"!! gives the value of a call that works", "!! works inside an expression, in a function that cannot fail", "!! works on a call with no value"}, abort: "panic: unwrapped error"},

	{dir: "char_escapes", src: `func code(char c) int {
    return int(c);
}

func main() {
    check(code('\0') == 0, "a char can be the zero byte");
    check(code('\n') == 10 && code('\r') == 13 && code('\t') == 9, "newline, carriage return and tab");
    check(code('\\') == 92, "an escaped backslash");
    check(code('\'') == 39, "an escaped single quote");
    check(code('\"') == 34, "an escaped double quote");
    check(code('\x41') == 65, "a hexadecimal escape");
    check(code('\x7f') == 127, "a lower case hexadecimal escape");
    check(code('a') == 97, "a plain char is unchanged");

    char[3] buf;
    buf[0] = 'h';
    buf[1] = 'i';
    buf[2] = '\0';
    check(buf[2] == '\0' && buf[0] != '\0', "the terminator can be stored in a buffer");
}
`, expected: []string{"a char can be the zero byte", "newline, carriage return and tab", "an escaped backslash", "an escaped single quote", "an escaped double quote", "a hexadecimal escape", "a lower case hexadecimal escape", "a plain char is unchanged", "the terminator can be stored in a buffer"}},

	{dir: "array_const_size", files: map[string]string{"limits.wsp": `public const Rows = 3;
public const Cells = Rows * 2;
`}, src: `import "limits";

const Width = 4;
const Area = Width * limits.Rows;

struct Board {
    int[Width] top;
    int[limits.Cells] cells;
}

func total(int[Width] row) int {
    int sum = 0;
    for _, v in row {
        sum += v;
    }
    return sum;
}

func main() {
    int[Width] a;
    int[Area] b;
    int[limits.Rows][Width] grid;
    int[Width + 1] c;
    Board board;

    check(len(a) == 4, "a local constant gives the size");
    check(len(b) == 12, "a constant built from an imported one gives the size");
    check(len(grid) == 3 && len(grid[0]) == 4, "constants size each dimension");
    check(len(c) == 5, "a constant expression gives the size");
    check(len(board.top) == 4 && len(board.cells) == 6, "a struct field can be sized by constants");

    a[0] = 1;
    a[1] = 2;
    a[2] = 3;
    a[3] = 4;
    check(total(a) == 10, "a parameter can be sized by a constant");
}
`, expected: []string{"a local constant gives the size", "a constant built from an imported one gives the size", "constants size each dimension", "a constant expression gives the size", "a struct field can be sized by constants", "a parameter can be sized by a constant"}},

	{dir: "const_modules", files: map[string]string{"flags.wsp": `public const Read = 1;
public const Write = 2;
public const Mask = Read | Write;
`, "tools.wsp": `import "flags";

public const Both = flags.Read | flags.Write;
public const Copy = flags.Write;
public const Offset = flags.Mask + 10;
`}, src: `import "flags";
import "tools";

const Local = tools.Copy * 4;

func main() {
    check(tools.Copy == 2, "a constant can be another module's constant");
    check(tools.Both == 3, "an expression of imported constants is constant");
    check(tools.Offset == 13, "a constant built from a constant that was itself built from another");
    check(Local == 8, "a constant in the importing module reads the re-exported one");
}
`, expected: []string{"a constant can be another module's constant", "an expression of imported constants is constant", "a constant built from a constant that was itself built from another", "a constant in the importing module reads the re-exported one"}},

	{dir: "global_modules", files: map[string]string{"store.wsp": `public int hits = 0;
public string tag = "a";
int hidden = 100;

public func bump() {
    hits++;
    hidden++;
}

public func read() int {
    return hits;
}

public func peek() int {
    return hidden;
}
`}, src: `import "store";

func main() {
    check(store.hits == 0, "imported global initial value");
    store.bump();
    check(store.hits == 1, "imported global changed by its module");
    store.hits = 10;
    check(store.read() == 10, "imported global written from outside is seen by the module");
    store.hits += 5;
    check(store.hits == 15 && store.read() == 15, "compound write on an imported global");
    check(store.tag == "a", "imported string global");
    store.tag = "b";
    check(store.tag == "b", "imported string global written");
    check(store.peek() == 101, "private global stays private");
    store.hits++;
    check(store.read() == 16, "increment on an imported global");
}
`, expected: []string{"imported global initial value", "imported global changed by its module", "imported global written from outside is seen by the module", "compound write on an imported global", "imported string global", "imported string global written", "private global stays private", "increment on an imported global"}},

	{dir: "global_pointer", src: `int slot = 5;
*int cell = null;
int[4] buffer;

func main() {
    check(cell == null, "global null pointer");
    cell = &slot;
    cell@ = 6;
    check(slot == 6, "write through a global pointer");
    cell = &buffer[2];
    cell@ = 7;
    check(buffer[2] == 7, "global pointer to a global array element");
}
`, expected: []string{"global null pointer", "write through a global pointer", "global pointer to a global array element"}},

	{dir: "string_pointer", src: `func main() {
    *int8 p = (*int8)("hello");
    check(p[0] == 104, "literal string data pointer first byte");
    check(p[4] == 111, "literal string data pointer last byte");
    check(p[5] == 0, "literal strings end with a zero byte");
    *char c = (*char)("hi");
    check(c[1] == 'i', "char pointer from a string");
    *uint8 u = (*uint8)("A");
    check(u[0] == 65, "uint8 pointer from a string");
    string s = "xyz";
    *int8 q = (*int8)(s);
    check(q[2] == 122, "pointer from a string variable");
    check((*int8)("z")@ == 122, "deref of a converted string");
}
`, expected: []string{"literal string data pointer first byte", "literal string data pointer last byte", "literal strings end with a zero byte", "char pointer from a string", "uint8 pointer from a string", "pointer from a string variable", "deref of a converted string"}},

	{dir: "module_const", files: map[string]string{"consts.wsp": `public const Limit = 7;
public const Name = "w";
public const On = true;
`}, src: `import "consts";

func main() {
    check(consts.Limit == 7, "an imported integer constant");
    check(consts.Name == "w", "an imported string constant");
    check(consts.On, "an imported bool constant");
    int[3] values = [1, 2, 3];
    check(values[consts.Limit - 6] == 2, "an imported constant as an index");
}
`, expected: []string{"an imported integer constant", "an imported string constant", "an imported bool constant", "an imported constant as an index"}},

	{dir: "param_span_after_comma", src: `func total(int scale, int[] values) int {
    int sum = 0;
    for _, v in values {
        sum += v;
    }
    return sum * scale;
}

func tag(string name, int[2] pair, bool on) int {
    return on ? pair[0] + len(name) : pair[1];
}

func main() {
    int[3] data = [1, 2, 3];
    check(total(2, data[0:3]) == 12, "a span parameter after another parameter");
    int[2] pair = [10, 20];
    check(tag("ab", pair, true) == 12, "an array parameter after another parameter");
}
`, expected: []string{"a span parameter after another parameter", "an array parameter after another parameter"}},

	{dir: "variadic_basic", src: `func sum(int ...values) int {
    int total;
    for i, v in values {
        total += v;
    }
    return total;
}

func count(int ...values) int {
    return len(values);
}

func first(int ...values) int {
    return values[0];
}

func withBase(int base, int ...rest) int {
    return base + sum(rest[0], rest[1]);
}

func main() {
    check(sum(1, 2, 3, 4) == 10, "a variadic function sums its arguments");
    check(sum() == 0, "a variadic function with no arguments");
    check(count(5, 6, 7) == 3, "len of a variadic parameter");
    check(first(7, 8) == 7, "indexing a variadic parameter");
    check(withBase(100, 1, 2) == 103, "fixed parameters before the variadic one");
}
`, expected: []string{"a variadic function sums its arguments", "a variadic function with no arguments", "len of a variadic parameter", "indexing a variadic parameter", "fixed parameters before the variadic one"}},

	{dir: "defer_blocks", src: `int trace = 0;

func mark(int n) {
    trace = trace * 10 + n;
}

func order() {
    defer mark(1);
    defer mark(2);
    defer {
        mark(3);
        mark(4);
    }
    mark(9);
}

func early(bool leave) int {
    defer mark(7);
    if leave {
        defer mark(8);
        return 5;
    }
    mark(6);
    return 6;
}

func frozen() int {
    int x = 5;
    defer x = 99;
    return x;
}

func loops() {
    for i in 1..3 {
        defer mark(i);
        if i == 2 {
            continue;
        }
        mark(0);
    }
}

func breaking() {
    loop {
        defer mark(1);
        defer mark(2);
        break;
    }
    mark(3);
}

func labeled() {
    :outer for 2 {
        defer mark(5);
        for 2 {
            defer mark(6);
            break outer;
        }
    }
}

func main() {
    trace = 0;
    order();
    check(trace == 93421, "deferred statements run in reverse order when the block ends");

    trace = 0;
    check(early(true) == 5 && trace == 87, "a return runs the defers of every block it leaves");
    trace = 0;
    check(early(false) == 6 && trace == 67, "the defers of the function run on the normal return");
    check(frozen() == 5, "the return value is computed before the defers run");

    trace = 0;
    loops();
    check(trace == 1203, "a defer in a loop body runs at the end of every iteration and on continue");
    trace = 0;
    breaking();
    check(trace == 213, "break runs the defers of the loop body");
    trace = 0;
    labeled();
    check(trace == 65, "a labeled break runs the defers of every block it leaves");
}
`, expected: []string{"deferred statements run in reverse order when the block ends", "a return runs the defers of every block it leaves", "the defers of the function run on the normal return", "the return value is computed before the defers run", "a defer in a loop body runs at the end of every iteration and on continue", "break runs the defers of the loop body", "a labeled break runs the defers of every block it leaves"}},

	{dir: "defer_on_error", src: `enum Fail {
    Ok,
    Boom,
}

int trace = 0;

func mark(int n) {
    trace = trace * 10 + n;
}

func inner(bool bad) !int {
    if bad {
        return Error(Fail.Boom);
    }
    return 1;
}

func fails(bool bad) !int {
    defer! mark(1);
    defer mark(2);
    if bad {
        return Error(Fail.Boom);
    }
    return 10;
}

func propagates(bool bad) !int {
    defer! mark(3);
    defer mark(4);
    int v = inner(bad)!;
    return v + 1;
}

func main() !int {
    trace = 0;
    int ok = fails(false)!;
    check(ok == 10 && trace == 2, "defer! does not run when the function succeeds");

    trace = 0;
    let _, _ = fails(true);
    check(trace == 21, "defer! runs together with defer when the function returns an error");

    trace = 0;
    let _, _ = propagates(true);
    check(trace == 43, "a propagated error runs defer! and defer");

    trace = 0;
    int z = propagates(false)!;
    check(z == 2 && trace == 4, "a propagating call that succeeds keeps defer! quiet");
    return 0;
}
`, expected: []string{"defer! does not run when the function succeeds", "defer! runs together with defer when the function returns an error", "a propagated error runs defer! and defer", "a propagating call that succeeds keeps defer! quiet"}},

	{dir: "enum_name", src: `enum Color {
    Red,
    Green,
}

enum Level {
    Low = 10,
    High = 20,
}

enum Mode {
    Fast,
    Slow,
}

func (Mode m) name() string {
    return "mine";
}

func main() {
    Color c = Color.Green;
    check(c.name() == "Green", "an enum value reports its member name");
    check(Color.Red.name() == "Red", "a member reports its own name");
    check(Level.High.name() == "High", "explicit values are matched by value");
    check(Color(7).name() == "?", "a value with no member reports a question mark");
    check(Mode.Fast.name() == "mine", "a name method written by hand replaces the generated one");
}
`, expected: []string{"an enum value reports its member name", "a member reports its own name", "explicit values are matched by value", "a value with no member reports a question mark", "a name method written by hand replaces the generated one"}},

	{dir: "any_variadic_float", luaOnly: true, src: `func kind(...args) int {
    switch let v = args[0] {
        case float:
            return 99;
        case int:
            return 1;
        default:
            return -1;
    }
}

func main() {
    check(kind(1.5) == 99, "case float matches a float");
    check(kind(2) == 1, "an int is not a float");
}
`, expected: []string{"case float matches a float", "an int is not a float"}},

	{dir: "any_variadic", src: `enum Color {
    Red,
    Green,
}

type Meters int;

func kind(int index, ...args) int {
    switch let v = args[index] {
        case int8:
            return 8;
        case Color:
            return 77;
        case Meters:
            return 88;
        case int:
            return v + 1000;
        case uint:
            return 2000;
        case string:
            return len(v);
        case bool:
            return v ? 1 : 0;
        case float:
            return 99;
        case char:
            return 55;
        default:
            return -1;
    }
}

func forward(...rest) int {
    return kind(0, rest...);
}

func count(...xs) int {
    return len(xs);
}

func sumInts(string label, ...xs) int {
    int total = 0;
    for _, x in xs {
        switch let v = x {
            case int:
                total += v;
            case string:
                total += len(v);
        }
    }
    return total;
}

func withPrefix(...rest) int {
    return kind(1, "prefix", rest...);
}

func onlyPrefix(...rest) int {
    return kind(0, "prefix", rest...);
}

func main() {
    int8 small = 5;
    uint big = 7;
    Meters m = Meters(3);
    check(kind(0, 5) == 1005, "an int literal matches case int and v is an int");
    check(kind(0, small) == 8, "a case for a sized type is exact and listed before the group");
    check(kind(0, big) == 2000, "case uint matches an unsigned integer");
    check(kind(0, "hello") == 5, "a string matches case string and v is a string");
    check(kind(0, true) == 1, "a bool matches case bool");
    check(kind(0, Color.Green) == 77, "an enum matches its own case before the int group");
    check(kind(0, m) == 88, "a nominal type matches only its own case");
    check(kind(0, 'x') == 55, "a char matches case char");
    check(kind(1, "a", 7) == 1007, "the second argument is read by index");
    check(forward(true) == 1, "... passes the variadic on");
    check(forward("abcd") == 4, "a forwarded string keeps its type");
    check(withPrefix(7) == 1007, "explicit arguments can come before the spread");
    check(onlyPrefix() == 6, "a spread of nothing leaves only the explicit arguments");
    check(count(1, "x", true) == 3, "len counts the arguments");
    check(count() == 0, "no arguments is an empty variadic");
    check(sumInts("sum", 1, "ab", 3, true) == 6, "for iterates the arguments and default is skipped");
}
`, expected: []string{"an int literal matches case int and v is an int", "a case for a sized type is exact and listed before the group", "case uint matches an unsigned integer", "a string matches case string and v is a string", "a bool matches case bool", "an enum matches its own case before the int group", "a nominal type matches only its own case", "a char matches case char", "the second argument is read by index", "... passes the variadic on", "a forwarded string keeps its type", "explicit arguments can come before the spread", "a spread of nothing leaves only the explicit arguments", "len counts the arguments", "no arguments is an empty variadic", "for iterates the arguments and default is skipped"}},

	{dir: "ptr_deref_postfix", src: `struct P {
    int x;
}

func first(*int p) *int {
    return p;
}

func main() {
    int x = 5;
    *int p = &x;
    **int pp = &p;
    P s = P{x: 3};
    *P sp = &s;

    p@ = 7;
    check(x == 7, "write through a pointer");
    p@ += 1;
    p@++;
    check(x == 9, "compound assignment and increment through a pointer");
    check(pp@@ == 9, "one @ per pointer level");
    pp@@ = 10;
    check(x == 10, "write through two levels");
    check(3 * p@ == 30, "multiplication next to a dereference");
    check(-p@ == -10, "unary minus before a dereference");
    sp@.x = 4;
    check(s.x == 4 && sp@.x == 4, "a field of the dereferenced struct");
    check(first(p)@ == 10, "dereference of a call");
    check(&p@ == p, "& is the inverse of @");
    check(&first(p)@ == p, "& of the dereference of a call");
    check(&pp@@ == p, "& of a double dereference");

    int[3] values = [1, 2, 3];
    *int base = &values[0];
    check((base + 2)@ == 3, "dereference of a parenthesized expression");
}
`, expected: []string{"write through a pointer", "compound assignment and increment through a pointer", "one @ per pointer level", "write through two levels", "multiplication next to a dereference", "unary minus before a dereference", "a field of the dereferenced struct", "dereference of a call", "& is the inverse of @", "& of the dereference of a call", "& of a double dereference", "dereference of a parenthesized expression"}},

	{dir: "ptr_convert_levels", x64Only: true, src: `func main() {
    int x = 5;
    *int p = &x;
    **int pp = (**int)(&p);
    check(pp@@ == 5, "a conversion to a double pointer");
    ***int ppp = (***int)(&pp);
    check(ppp@@@ == 5, "a conversion to a triple pointer");

    char a = 5;
    uint8 b = 3 * (*uint8)(&a)@;
    check(b == 15, "a conversion next to a multiplication and a dereference");

    *int8 raw = (*int8)(p);
    **int8 rawpp = (**int8)(&raw);
    check(rawpp@@ == 5, "a double pointer to bytes");
}
`, expected: []string{"a conversion to a double pointer", "a conversion to a triple pointer", "a conversion next to a multiplication and a dereference", "a double pointer to bytes"}},

	{dir: "native_bindings", x64Only: true, src: `struct P {
    int x;
    int y;
}

int counter = 5;
P origin = P{x: 3, y: 4};

func main() {
    int a = 40;
    int b = 2;
    int r = 0;
    native x64(in a, b, out r) "mov rax, {a}; add rax, {b}; mov {r}, rax";
    check(r == 42, "bindings in sequence share their mode");

    int c = 41;
    native x64(inout c) "add {c}, 1";
    check(c == 42, "inout reads and writes");

    int out = 5;
    int d = 0;
    native x64(in out, out d) "mov rax, {out}; mov {d}, rax";
    check(d == 5, "a variable can be named out");

    P p = P{x: 10, y: 20};
    int sum = 0;
    native x64(in p, out sum) "mov rax, {p.y}; add rax, {p.x}; mov {sum}, rax; mov {p.x}, 99";
    check(sum == 30 && p.x == 99 && p.y == 20, "struct fields are operands");

    int[4] values = [1, 2, 3, 4];
    int third = 0;
    native x64(in values, out third) "lea rax, {&values}; mov rax, [rax+16]; mov {third}, rax";
    check(third == 3, "{&name} is the address of any variable");

    string s = "hey";
    int n = 0;
    native x64(in s, out n) "mov rax, {s.len}; mov {n}, rax";
    check(n == 3, "string parts are operands");

    int before = 0;
    native x64(in counter, out before) "mov rax, {counter}; mov {before}, rax; add {counter}, 1";
    check(before == 5 && counter == 6, "a global is an operand");

    int gy = 0;
    native x64(in origin, out gy) "mov rax, {origin.y}; mov {gy}, rax; mov {origin.x}, 30";
    check(gy == 4 && origin.x == 30, "a global struct has fields");
}
`, expected: []string{"bindings in sequence share their mode", "inout reads and writes", "a variable can be named out", "struct fields are operands", "{&name} is the address of any variable", "string parts are operands", "a global is an operand", "a global struct has fields"}},

	{dir: "native_raw_block", x64Only: true, src: "func main() {\n" +
		"    int r = 0;\n" +
		"    native x64(out r) `\n" +
		"        ; the answer\n" +
		"        mov rcx, 3\n" +
		"        xor rax, rax\n" +
		"    .again:\n" +
		"        add rax, 14   ; a comment after code\n" +
		"        dec rcx\n" +
		"        jnz .again\n" +
		"        mov {r}, rax\n" +
		"    `;\n" +
		"    check(r == 42, \"a backtick block goes to the assembler as written\");\n" +
		"}\n", expected: []string{"a backtick block goes to the assembler as written"}},

	{dir: "interop_extern", x64Only: true, files: map[string]string{"lib.c": `int add(int a, int b) { return a + b; }
long mul64(long a, long b) { return a * b; }
int sum_bytes(const char *p, int n) {
    int t = 0;
    for (int i = 0; i < n; i++) t += (unsigned char)p[i];
    return t;
}
int twice(int x) { return 2 * x; }
unsigned char low_byte(unsigned int v) { return (unsigned char)(v & 0xff); }
int count_calls(void) { static int n = 0; return ++n; }
`}, src: `extern func add(int32 a, int32 b) int32;
extern func mul64(int64 a, int64 b) int64;
extern func sum_bytes(*int8 p, int32 n) int32;
extern func twice(int32 x) int32 as c_twice;
extern func low_byte(uint32 v) uint8;
extern func count_calls() int32;

func main() ! {
    check(add(2, 3) == 5, "extern function with two arguments");
    check(mul64(6, 7) == 42, "extern function with 64-bit arguments");
    *int8 b = malloc(4)!;
    b[0] = 1;
    b[1] = 2;
    b[2] = 3;
    b[3] = 4;
    check(sum_bytes(b, 4) == 10, "extern function taking a pointer");
    free(b);
    check(c_twice(21) == 42, "extern function renamed with as");
    check(low_byte(0x1234) == 0x34, "extern function returning uint8");
    count_calls();
    check(count_calls() == 2, "C keeps its own static state");
    int8 small = 7;
    check(add(int32(small), 1) == 8, "narrow argument widened for C");
    check(sum_bytes((*int8)("abc"), 3) == 294, "pointer to the bytes of a string");
}
`},

	{dir: "interop_export", x64Only: true, cmain: true, files: map[string]string{"lib.wsp": `public func add(int32 a, int32 b) int32 as wisp_add {
    return a + b;
}

public func scale(int64 v, int64 k) int64 as wisp_scale => v * k;

func local() int32 {
    return 1;
}

public func next(int32 a) int32 as wisp_next {
    return add(a, local());
}

public func plain(int32 a) int32 {
    return a;
}
`, "main.c": `#include <stdio.h>

extern int wisp_add(int, int);
extern long wisp_scale(long, long);
extern int wisp_next(int);

int main(void) {
    printf("%d %ld %d\n", wisp_add(2, 3), wisp_scale(6, 7), wisp_next(9));
    return 0;
}
`}, expected: []string{"5 42 10"}},

	{dir: "interop_libc_puts", x64Only: true, libc: true, src: `extern func puts(*int8 s) int32;
extern func strlen(*int8 s) int64;

func main() {
    check(strlen((*int8)("hello")) == 5, "libc strlen on a string literal");
    puts((*int8)("hi"));
}
`, expected: []string{"libc strlen on a string literal", "hi"}},

	{dir: "interop_libc_printf", x64Only: true, libc: true, src: `extern func printf(*int8 fmt, ...) int32;

func main() int {
    string name = "wisp";
    printf((*int8)("%d %s %c|%ld\n"), 42, (*int8)(name), 'x', 5000000000);
    printf((*int8)("no args\n"));
    printf((*int8)("%d %d %d %d %d %d %d\n"), 1, 2, 3, 4, 5, 6, 7);
    return 0;
}
`, expected: []string{"42 wisp x|5000000000", "no args", "1 2 3 4 5 6 7"}},

	{dir: "interop_libc_main_error", x64Only: true, libc: true, exit: 3, src: `extern func puts(*int8 s) int32;

enum E {
    Ok,
    Other,
    Again,
    Bad
}

func fail() !int {
    return Error(E.Bad, "boom");
}

func main() !int {
    puts((*int8)("hi"));
    return fail()!;
}
`, expected: []string{"hi"}, abort: "error[main.E]: boom"},

	{dir: "method_same_name", src: `struct A {
    int x;
}

struct B {
    int y;
}

func (*A a) get() int {
    return a.x;
}

func (*B b) get() int {
    return b.y;
}

func (*A a) twice() int {
    return a.get() * 2;
}

func get() int {
    return 100;
}

func main() {
    A a = A{x: 1};
    B b = B{y: 2};
    check(a.get() == 1, "a method on the first struct");
    check(b.get() == 2, "a method with the same name on another struct");
    check(a.twice() == 2, "a method calling a method of its own type");
    check(get() == 100, "a function with the same name as the methods");
}
`, expected: []string{"a method on the first struct", "a method with the same name on another struct", "a method calling a method of its own type", "a function with the same name as the methods"}},

	{dir: "abort_main_error", exit: 2, src: `enum MathError {
    a,
    b,
    DivByZero
}

func divide(int a, int b) !int {
    if b == 0 {
        return Error(MathError.DivByZero, "division by zero");
    }
    return a / b;
}

func main() ! {
    check(divide(8, 2)! == 4, "mainerr before the failure");
    divide(4, 0)!;
    check(false, "mainerr never printed");
}
`, expected: []string{"mainerr before the failure"}, abort: "error[main.MathError]: division by zero"},

	{dir: "abort_main_error_int", exit: 1, src: `enum Kind {
    Bad = 1
}

func fail() !int {
    return Error(Kind.Bad, "bad thing");
}

func main() !int {
    check(true, "mainerrint before the failure");
    return fail()!;
}
`, expected: []string{"mainerrint before the failure"}, abort: "error[main.Kind]: bad thing"},
}
