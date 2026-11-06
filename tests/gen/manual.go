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
    uint64 x = 0;
    x -= 1;
}
`},

	{dir: "big_literal_wide", x64Only: true, src: `func id(int64 x) int64 {
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
    char c = 65;
    c += 1;
    check(c == 66, "alias char arithmetic");
    char d = 255;
    d += 1;
    check(d == 0, "alias char wraps like uint8");
}
`},

	{dir: "alias_types_wide", x64Only: true, src: `func main() {
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
    uint8 g = uint8(200);
    check(g == 200, "castlit constant that fits");
    int8 h = int8(0 + 100);
    check(h == 100, "castlit constant expression");
}
`},

	{dir: "cast_literal_wide", x64Only: true, src: `func main() {
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
	{dir: "array_to_slice", src: `struct Box {
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

}
`, expected: []string{"an array variable passes as a slice", "the slice is a view of the array", "an array initialises a slice variable", "an array is assigned to a slice", "an array field passes as a slice", "an array element passes as a slice", "an explicit slice still works", "an array literal passes as a slice"}},
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
	{dir: "empty_literal", src: `import "std/mem";
struct Holder {
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

    mem.free((*char)(a), (*char)(b));
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
}
`},

	{dir: "view_semantics", x64Only: true, src: `import "std/fmt";
import "std/mem";

char[64] scratch;
char[64] other;

func main() ! {
    char[3] word = ['a', 'b', 'c'];
    string seen = string(word);
    word[0] = 'x';
    check(seen == "xbc", "a char array converts to a string view of itself");

    *int8 b = mem.alloc(8)!;
    b[0] = 104;
    b[1] = 105;
    string s = string(b[0:2]);
    b[1] = 111;
    check(s == "ho", "string is a view of the buffer");
    mem.free(b);

    int x = 5;
    *int p = &x;
    let n, err = fmt.sprintf(scratch[:], "{}", p);
    check(err == null && n > 2 && string(scratch[0:2]) == "0x", "a pointer starts with 0x");
    let m, err2 = fmt.sprintf(other[:], "{p}", p);
    check(err2 == null && m == n && string(scratch[0:n]) == string(other[0:m]), "p is the default for a pointer");
}
`, expected: []string{"a char array converts to a string view of itself", "string is a view of the buffer", "a pointer starts with 0x", "p is the default for a pointer"}},

	{dir: "float_basic", src: `func main() {
    float x = 2.5;
    check(int(x) == 2, "a float literal truncates toward zero");
    check(int(0.0 - 2.9) == -2, "a negative float truncates toward zero");

    float32 small = 1.5;
    float widened = small;
    check(int(widened * 4.0) == 6, "float32 widens to float");
    float32 narrowed = float32(widened);
    check(int(narrowed * 2.0) == 3, "float narrows to float32");

    int n = 7;
    float fromInt = float(n);
    check(int(fromInt) == 7, "an int converts to a float");
    int8 minus = -3;
    check(int(float(minus)) == -3, "a negative int8 converts to a float");
    uint8 high = 200;
    check(int(float(high)) == 200, "a uint8 converts to a float");
    int64 big = 4000000000;
    check(int(float(big)) == 4000000000, "an int64 converts to a float and back");

    float copy = x;
    check(int(copy) == 2, "a float variable is copied");
}
`, expected: []string{"a float literal truncates toward zero", "a negative float truncates toward zero", "float32 widens to float", "float narrows to float32", "an int converts to a float", "a negative int8 converts to a float", "a uint8 converts to a float", "an int64 converts to a float and back", "a float variable is copied"}},

	{dir: "float_arith", src: `func main() {
    float a = 2.5;
    float b = a * 2.0 + 1.0;
    check(int(b) == 6, "multiply then add");
    check(int(b - 1.0) == 5, "subtract");
    check(int(7.0 / 2.0 * 2.0) == 7, "divide keeps the fraction");
    check(int(1.0 / 4.0 * 100.0) == 25, "a fraction is not truncated before the next operation");

    float x = 10.0;
    x += 2.5;
    x *= 2.0;
    x /= 5.0;
    x -= 0.5;
    check(int(x) == 4, "compound assignment on a float");

    float32 f = 1.5;
    float32 g = f * 4.0;
    check(int(g) == 6, "float32 arithmetic");
    check(int(float(g) / 2.0) == 3, "float32 widened into a float divide");

    check(int(a * a * a) == 15, "chained multiplication");
    check(int((a + 0.5) * (a - 0.5)) == 6, "parenthesised operands");
    float zero = 0.0;
    check(int(1.0 / 3.0 * 3.0 + zero) == 1, "no panic is needed to divide by a float");
}
`, expected: []string{"multiply then add", "subtract", "divide keeps the fraction", "a fraction is not truncated before the next operation", "compound assignment on a float", "float32 arithmetic", "float32 widened into a float divide", "chained multiplication", "parenthesised operands", "no panic is needed to divide by a float"}},

	{dir: "float_compare", src: `func main() {
    float a = 1.5;
    float b = 2.5;
    check(a < b, "less");
    check(a <= a, "less or equal when equal");
    check(a <= b, "less or equal when less");
    check(b > a, "greater");
    check(b >= b, "greater or equal when equal");
    check(b >= a, "greater or equal when greater");
    check(a == a, "equal");
    check(a != b, "not equal");
    checkEq(a > b, false, "a smaller number is not greater");
    checkEq(b < a, false, "a bigger number is not less");
    checkEq(a == b, false, "different numbers are not equal");
    checkEq(a >= b, false, "a smaller number is not greater or equal");

    float32 s = 1.5;
    float32 u = 2.5;
    check(s < u, "float32 less");
    check(u >= s, "float32 greater or equal");
    check(s == s, "float32 equal");
    check(s != u, "float32 not equal");

    float zero = 0.0;
    float nan = zero / zero;
    checkEq(nan == nan, false, "NaN is not equal to itself");
    check(nan != nan, "NaN is different from itself");
    checkEq(nan < 1.0, false, "NaN is not less");
    checkEq(nan > 1.0, false, "NaN is not greater");
    checkEq(nan <= nan, false, "NaN is not less or equal");
    checkEq(nan >= nan, false, "NaN is not greater or equal");
    checkEq(1.0 < nan, false, "nothing is less than NaN");
    checkEq(1.0 > nan, false, "nothing is greater than NaN");
    check(1.0 != nan, "nothing is equal to NaN");

    int count = 0;
    if a < b {
        count++;
    }
    if b < a {
        count += 10;
    }
    check(count == 1, "a float comparison drives an if");
}
`, expected: []string{"less", "less or equal when equal", "less or equal when less", "greater", "greater or equal when equal", "greater or equal when greater", "equal", "not equal", "a smaller number is not greater", "a bigger number is not less", "different numbers are not equal", "a smaller number is not greater or equal", "float32 less", "float32 greater or equal", "float32 equal", "float32 not equal", "NaN is not equal to itself", "NaN is different from itself", "NaN is not less", "NaN is not greater", "NaN is not less or equal", "NaN is not greater or equal", "nothing is less than NaN", "nothing is greater than NaN", "nothing is equal to NaN", "a float comparison drives an if"}},

	{dir: "float_abi", src: `struct Vec {
    float x;
    float y;
}

struct Tag {
    int id;
    float weight;
}

struct Small {
    float32 a;
    float32 b;
}

struct Wide {
    float a;
    float b;
    float c;
}

func add(float a, float b) int {
    return int(a + b);
}

func mix(int n, float x, int m, float y) int {
    return n + m + int(x + y);
}

func small(float32 a, float32 b) int {
    return int(a * b);
}

func many(float a, float b, float c, float d, float e, float f, float g, float h, float i, float j) int {
    return int(a + b + c + d + e + f + g + h + i + j);
}

func half(float x) float {
    return x / 2.0;
}

func twice(float a) (float, int) {
    return a * 2.0, int(a);
}

func swap(float a, float b) (float, float) {
    return b, a;
}

func fall(int n) !float {
    return float(n) * 1.5;
}

func mixed(int n, float x) (int, float, int) {
    return n, x, n + 1;
}

func three(float a) (float, float, float) {
    return a, a * 2.0, a * 3.0;
}

func length(Vec v) int {
    return int(v.x + v.y);
}

func scale(Vec v, float k) Vec {
    return Vec{x: v.x * k, y: v.y * k};
}

func describe(Tag t) float {
    return t.weight * float(t.id);
}

func make(int id, float w) Tag {
    return Tag{id: id, weight: w};
}

func shrink(Small s) Small {
    return Small{a: s.a / 2.0, b: s.b / 2.0};
}

func total(Wide w) int {
    return int(w.a + w.b + w.c);
}

func main() ! {
    check(add(2.5, 3.5) == 6, "two floats arrive in xmm0 and xmm1");
    check(mix(1, 2.5, 2, 3.5) == 9, "integer and float registers are counted apart");
    check(small(1.5, 4.0) == 6, "float32 arguments");
    check(many(1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0) == 10, "more floats than registers spill to the stack");

    check(int(half(5.0) * 10.0) == 25, "a float result comes back in xmm0");
    let d, i = twice(2.5);
    check(int(d) == 5 && i == 2, "a float and an int come back together");
    let s1, s2 = swap(1.5, 4.5);
    check(int(s1) == 4 && int(s2) == 1, "two floats come back in xmm0 and xmm1 without crossing");
    float f = fall(2)!;
    check(int(f) == 3, "a fallible function returns a float");
    let a, b, c = mixed(7, 2.5);
    check(a == 7 && int(b) == 2 && c == 8, "two ints and a float come back in rax, rdx and xmm0");
    let t1, t2, t3 = three(1.0);
    check(int(t1) == 1 && int(t2) == 2 && int(t3) == 3, "three floats come back through memory");

    check(length(Vec{x: 2.5, y: 3.5}) == 6, "a struct of two floats is passed in two xmm registers");
    Vec scaled = scale(Vec{x: 1.5, y: 2.5}, 2.0);
    check(int(scaled.x) == 3 && int(scaled.y) == 5, "a struct of two floats is returned in xmm0 and xmm1");
    check(int(describe(Tag{id: 3, weight: 2.5})) == 7, "a struct with an int and a float is passed in one of each");
    Tag tag = make(4, 1.5);
    check(tag.id == 4 && int(tag.weight * 10.0) == 15, "a struct with an int and a float is returned in rax and xmm0");
    Small sm = shrink(Small{a: 3.0, b: 5.0});
    check(int(sm.a * 10.0) == 15 && int(sm.b * 10.0) == 25, "two float32 fields share one xmm register");
    check(total(Wide{a: 1.0, b: 2.0, c: 3.0}) == 6, "a struct larger than sixteen bytes goes through memory");
}
`, expected: []string{"two floats arrive in xmm0 and xmm1", "integer and float registers are counted apart", "float32 arguments", "more floats than registers spill to the stack", "a float result comes back in xmm0", "a float and an int come back together", "two floats come back in xmm0 and xmm1 without crossing", "a fallible function returns a float", "two ints and a float come back in rax, rdx and xmm0", "three floats come back through memory", "a struct of two floats is passed in two xmm registers", "a struct of two floats is returned in xmm0 and xmm1", "a struct with an int and a float is passed in one of each", "a struct with an int and a float is returned in rax and xmm0", "two float32 fields share one xmm register", "a struct larger than sixteen bytes goes through memory"}},

	{dir: "float_uint64", x64Only: true, src: `func main() {
    uint64 big = 18446744073709551615;
    float f = float(big);
    check(f > 18000000000000000000.0 && f < 19000000000000000000.0, "the largest uint64 becomes a float near two to the 64");

    uint64 half = 9223372036854775808;
    check(float(half) == 9223372036854775808.0, "two to the 63 converts exactly");

    float up = 9223372036854775808.0;
    check(uint64(up) == 9223372036854775808, "a float of two to the 63 converts back");
    check(uint64(18446744073709549568.0) == 18446744073709549568, "the largest double below two to the 64 converts back");

    uint64 small = 5;
    check(float(small) == 5.0, "a small uint64 converts to a float");
    check(uint64(7.9) == 7, "a float converts to a uint64 by truncating");

    float32 narrow = float32(big);
    check(narrow > 18000000000000000000.0 && narrow < 19000000000000000000.0, "a uint64 converts to a float32");

    uint64 odd = 9223372036854775809;
    check(float(odd) == 9223372036854775808.0, "the dropped bit rounds to even");
    uint64 rounded = 18446744073709551614;
    check(float(rounded) == 18446744073709551616.0, "a value near the top rounds up");

    uint64 mid = 4611686018427387904;
    check(uint64(float(mid)) == mid, "a uint64 below two to the 63 makes the round trip");

    float32 sm = 3.5;
    check(uint64(sm) == 3, "a float32 converts to a uint64");

    uint32 word = 4000000000;
    check(float(word) == 4000000000.0, "a uint32 converts to a float");
    check(uint32(float(word)) == word, "a float converts to a uint32");
}
`, expected: []string{"the largest uint64 becomes a float near two to the 64", "two to the 63 converts exactly", "a float of two to the 63 converts back", "the largest double below two to the 64 converts back", "a small uint64 converts to a float", "a float converts to a uint64 by truncating", "a uint64 converts to a float32", "the dropped bit rounds to even", "a value near the top rounds up", "a uint64 below two to the 63 makes the round trip", "a float32 converts to a uint64", "a uint32 converts to a float", "a float converts to a uint32"}},

	{dir: "float_any", src: `func sum(...args) int {
    int total;
    for _, arg in args {
        switch let v = arg {
            case float32:
                total += int(v * 10.0);
            case float64:
                total += int(v);
            case int:
                total += v;
        }
    }
    return total;
}

func typed(float ...xs) int {
    float total = 0.0;
    for _, x in xs {
        total += x;
    }
    return int(total);
}

func forward(...args) int {
    return sum(args...);
}

func half(float x) float {
    return x / 2.0;
}

func main() {
    float32 a = 1.5;
    float b = 4.0;
    float neg = 0.0 - 2.5;
    check(sum(2.5, a, 7, b) == 28, "floats and an int in a variadic of any");
    check(sum(b * 2.0, a * 2.0, half(b), -b, float32(b) + a, 3) == 94, "a computed float is boxed");
    check(sum(neg) == -2, "a negative float keeps its sign in the box");
    check(forward(a, 2.5, 3) == 20, "floats are forwarded with a spread");
    check(typed(1.5, 2.5, half(8.0)) == 8, "a variadic of float takes computed floats");
}
`, expected: []string{"floats and an int in a variadic of any", "a computed float is boxed", "a negative float keeps its sign in the box", "floats are forwarded with a spread", "a variadic of float takes computed floats"}},

	{dir: "float_exponent", src: `func main() {
    float a = 1e3;
    float b = 2.5e-3;
    float c = 1E+2;
    float d = .5e1;
    float e = 1_0e1_0;
    float32 f = 3e2;
    float big = 1.7976931348623157e308;
    float tiny = 5e-324;
    check(a == 1000.0, "an exponent without a point is a float");
    check(b * 1000.0 > 2.4 && b * 1000.0 < 2.6, "a negative exponent");
    check(c == 100.0, "an upper case E and a plus sign");
    check(d == 5.0, "an exponent after a leading point");
    check(e == 100000000000.0, "underscores in the digits and in the exponent");
    check(f == 300.0, "an exponent literal as a float32");
    check(big > 1e308 && big / 2.0 < big, "the largest float64");
    check(tiny > 0.0 && tiny / 2.0 == 0.0, "the smallest subnormal float64");
    int x = 3;
    int e2 = 2;
    check(x > e2, "an identifier that starts with e is not an exponent");
}
`, expected: []string{"an exponent without a point is a float", "a negative exponent", "an upper case E and a plus sign", "an exponent after a leading point", "underscores in the digits and in the exponent", "an exponent literal as a float32", "the largest float64", "the smallest subnormal float64", "an identifier that starts with e is not an exponent"}},

	{dir: "float_math", src: `import "std/math";

func main() {
    check(math.floor(2.7) == 2.0 && math.floor(-2.5) == -3.0, "floor goes down");
    check(math.ceil(2.1) == 3.0 && math.ceil(-2.5) == -2.0, "ceil goes up");
    check(math.trunc(2.9) == 2.0 && math.trunc(-2.9) == -2.0, "trunc goes toward zero");
    check(math.round(2.5) == 3.0 && math.round(-2.5) == -3.0, "round goes away from zero on a half");
    check(math.round(0.5) == 1.0 && math.round(-0.5) == -1.0, "round of one half");
    check(math.round(0.49999999999999994) == 0.0, "round of the largest number below one half");
    check(math.round(2.4) == 2.0 && math.round(-2.4) == -2.0, "round below a half");
    check(math.round(4503599627370497.0) == 4503599627370497.0, "round leaves a large float alone");
    check(math.roundEven(2.5) == 2.0 && math.roundEven(3.5) == 4.0, "roundEven goes to the even neighbour");
    check(math.roundEven(-2.5) == -2.0 && math.roundEven(2.6) == 3.0, "roundEven of a negative half and a plain case");

    check(math.fsqrt(16.0) == 4.0 && math.fsqrt(0.0) == 0.0, "square root");
    check(math.fsqrt(2.0) * math.fsqrt(2.0) > 1.9999999999999 && math.fsqrt(2.0) * math.fsqrt(2.0) < 2.0000000000001, "square root of two");
    check(math.isNaN(math.fsqrt(-1.0)), "square root of a negative is NaN");
    check(math.isInf(math.fsqrt(math.inf(1)), 1), "square root of infinity");

    float nan = math.nan();
    check(math.isNaN(nan) && !math.isNaN(1.0), "isNaN");
    check(nan != nan, "NaN is different from itself");
    check(math.isInf(math.inf(1), 1) && !math.isInf(math.inf(1), -1) && math.isInf(math.inf(-1), -1), "isInf with a sign");
    check(math.isInf(math.inf(-1), 0) && !math.isInf(1.0, 0), "isInf of either sign");
    check(math.isFinite(1.0) && !math.isFinite(nan) && !math.isFinite(math.inf(1)), "isFinite");
    check(math.isInteger(4.0) && !math.isInteger(4.5) && !math.isInteger(math.inf(1)), "isInteger");

    check(math.fabs(-3.5) == 3.5 && math.fabs(3.5) == 3.5, "fabs");
    check(!math.signbit(math.fabs(-0.0)) && math.signbit(-0.0) && !math.signbit(0.0), "fabs and signbit on zero");
    check(math.copysign(3.0, -1.0) == -3.0 && math.copysign(-3.0, 1.0) == 3.0, "copysign");

    check(math.fmin(1.0, 2.0) == 1.0 && math.fmax(1.0, 2.0) == 2.0, "fmin and fmax");
    check(math.isNaN(math.fmin(1.0, nan)) && math.isNaN(math.fmax(nan, 1.0)), "fmin and fmax with a NaN give NaN");
    check(math.signbit(math.fmin(0.0, -0.0)) && !math.signbit(math.fmax(-0.0, 0.0)), "negative zero is below positive zero");
    check(math.fclamp(5.5, 1.0, 3.0) == 3.0 && math.fclamp(-1.0, 1.0, 3.0) == 1.0 && math.fclamp(2.0, 1.0, 3.0) == 2.0, "fclamp");

    float whole, fraction = math.modf(3.75);
    check(whole == 3.0 && fraction == 0.75, "modf of a positive number");
    float negWhole, negFraction = math.modf(-3.75);
    check(negWhole == -3.0 && negFraction == -0.75, "modf of a negative number");
}
`, expected: []string{"floor goes down", "ceil goes up", "trunc goes toward zero", "round goes away from zero on a half", "round of one half", "round of the largest number below one half", "round below a half", "round leaves a large float alone", "roundEven goes to the even neighbour", "roundEven of a negative half and a plain case", "square root", "square root of two", "square root of a negative is NaN", "square root of infinity", "isNaN", "NaN is different from itself", "isInf with a sign", "isInf of either sign", "isFinite", "isInteger", "fabs", "fabs and signbit on zero", "copysign", "fmin and fmax", "fmin and fmax with a NaN give NaN", "negative zero is below positive zero", "fclamp", "modf of a positive number", "modf of a negative number"}},

	{dir: "float_globals", src: `float g = 1;
float[3] c = [1.5, 2, 3.5e-3];
float32 h = 3;
float64 big = 1.7976931348623157e308;
float[2] negative = [-2.5, 0.0];

func main() {
    check(g == 1.0, "an integer literal initializes a float global");
    check(c[0] == 1.5 && c[1] == 2.0 && c[2] > 0.0034 && c[2] < 0.0036, "a float array global");
    check(h == 3.0, "a float32 global");
    check(big > 1e308, "the largest float as a global");
    check(negative[0] == -2.5 && negative[1] == 0.0, "negative and zero elements");
}
`, expected: []string{"an integer literal initializes a float global", "a float array global", "a float32 global", "the largest float as a global", "negative and zero elements"}},

	{dir: "native_consts", x64Only: true, files: map[string]string{"limits/limits.wsp": `public const {
    Base = 40;
    Extra = 2;
}
`, "main.wsp": `import "limits";

const {
    Greeting = "hey";
    Bias = -1;
    Enabled = true;
}

func check(bool ok, string name) {
    if ok {
        emit(name);
    } else {
        emit("FAIL");
        emit(name);
    }
}

func main() {
    int sum;
    native linux_x64(out sum) "mov {sum}, {limits.Base}; add {sum}, {limits.Extra}";
    check(sum == 42, "a constant of another module is an immediate");

    int bias;
    native linux_x64(out bias) "mov {bias}, {Bias}";
    check(bias == -1, "a negative constant keeps its sign");

    int on;
    native linux_x64(out on) "mov {on}, {Enabled}";
    check(on == 1, "a bool constant is 1 or 0");

    int n;
    *char p;
    native linux_x64(out n, p) "mov {n}, {Greeting.len}; lea rax, [rel {Greeting.ptr}]; mov {p}, rax";
    check(n == 3, "the length of a string constant");
    check(string(p[:n]) == "hey", "the address of a string constant");
}
`}, expected: []string{"a constant of another module is an immediate", "a negative constant keeps its sign", "a bool constant is 1 or 0", "the length of a string constant", "the address of a string constant"}},

	{dir: "const_group", files: map[string]string{"limits/limits.wsp": `public const {
    Rows = 4;
    Cols = Rows * 2;
}

internal const int8 {
    Small = 100;
    Tiny = 5;
}

public func total() int {
    return Small + Tiny;
}
`, "main.wsp": `import "limits";

const {
    Limit = 10;
    Double = Limit * 2;
}

const uint8 {
    Mask = 0xF0;
    Low = 0x0F;
}

func check(bool ok, string name) {
    if ok {
        emit(name);
    } else {
        emit("FAIL");
        emit(name);
    }
}

func main() {
    int[Double] cells;
    check(Limit == 10 && Double == 20, "members of a group can use the ones before them");
    check(len(cells) == 20, "a group member is a constant expression");
    check(Mask + Low == 255, "the type of a group applies to every member");
    uint8 m = Mask;
    check(m == 240, "a typed group member converts like a typed const");
    check(limits.Rows == 4 && limits.Cols == 8, "public makes every member of a group public");
    check(limits.total() == 105, "internal members stay usable inside their folder");
}
`}, expected: []string{"members of a group can use the ones before them", "a group member is a constant expression", "the type of a group applies to every member", "a typed group member converts like a typed const", "public makes every member of a group public", "internal members stay usable inside their folder"}},

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
    int x = 100;
    x /= c;
    check(x == 25, "divreg compound division");
    x %= 7;
    check(x == 4, "divreg compound remainder");
}
`},

	{dir: "divmod_registers_wide", x64Only: true, src: `func idiv(int a, int b) int {
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

	{dir: "trace_nested", src: `func div(int a, int b) int {
    return a / b;
}

func compute(int x) int {
    int c = div(10, x);
    return c;
}

func outer(int x) int {
    return compute(x) + 1;
}

func main() {
    check(outer(5) == 3, "trace before the failing call");
    int r = outer(0);
    check(r == 0, "trace never printed");
}
`, expected: []string{"trace before the failing call"}, abort: ": in main.compute", x64Only: true},

	{dir: "trace_outer", src: `func div(int a, int b) int {
    return a / b;
}

func compute(int x) int {
    int c = div(10, x);
    return c;
}

func outer(int x) int {
    return compute(x) + 1;
}

func main() {
    check(outer(5) == 3, "trace outer before the failing call");
    int r = outer(0);
    check(r == 0, "trace outer never printed");
}
`, expected: []string{"trace outer before the failing call"}, abort: ": in main.outer", x64Only: true},

	{dir: "trace_main", src: `func div(int a, int b) int {
    return a / b;
}

func main() {
    check(div(4, 2) == 2, "trace main before the failing call");
    int r = div(1, 0);
    check(r == 0, "trace main never printed");
}
`, expected: []string{"trace main before the failing call"}, abort: ": in main.main", x64Only: true},

	{dir: "trace_method", src: `struct Counter {
    int n;
}

func (*Counter c) share(int by) int {
    return c.n / by;
}

func (*Counter c) split(int by) int {
    return c.share(by);
}

func main() {
    Counter c = Counter{n: 8};
    check(c.split(2) == 4, "trace method before the failing call");
    int r = c.split(0);
    check(r == 0, "trace method never printed");
}
`, expected: []string{"trace method before the failing call"}, abort: ": in main.Counter.split", x64Only: true},

	{dir: "trace_unwrap", src: `enum Fail {
    Bad = 1
}

func risky(int n) !int {
    if n < 0 {
        return Error(Fail.Bad, "negative input");
    }
    return n;
}

func settle(int n) int {
    return risky(n)!!;
}

func main() {
    check(settle(2) == 2, "trace unwrap before the failing call");
    int r = settle(-1);
    check(r == 0, "trace unwrap never printed");
}
`, expected: []string{"trace unwrap before the failing call"}, abort: ": in main.main", x64Only: true},

	{dir: "trace_lambda", src: `func apply(Step f, int v) int {
    return f(v);
}

type Step (int) => int;

func main() {
    Step half = (int v) => 10 / v;
    check(apply(half, 5) == 2, "trace lambda before the failing call");
    int r = apply(half, 0);
    check(r == 0, "trace lambda never printed");
}
`, expected: []string{"trace lambda before the failing call"}, abort: ": in main.apply", x64Only: true},

	{dir: "trace_deep", src: `func down(int n) int {
    if n == 0 {
        return 1 / n;
    }
    return down(n - 1) + 1;
}

func main() {
    check(true, "trace deep before the failing call");
    int r = down(60);
    check(r == 0, "trace deep never printed");
}
`, expected: []string{"trace deep before the failing call"}, abort: "more frames", x64Only: true},

	{dir: "abort_double_free", src: `import "std/mem";

func main() {
    *char p = mem.alloc(48) ?? null;
    *char q = mem.alloc(48) ?? null;
    check(p != null && q != null, "double free before the second free");
    mem.free(p);
    mem.free(q);
    mem.free(null);
    *char r = mem.alloc(48) ?? null;
    check(r != null, "a freed block is reused without a complaint");
    mem.free(r);
    mem.free(r);
    check(true, "double free never printed");
}
`, expected: []string{"double free before the second free", "a freed block is reused without a complaint"}, abort: "panic: double free", x64Only: true},

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

	{dir: "abort_div_zero_uint64", x64Only: true, src: `func main() {
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
    check(min64 % neg64 == 0, "minneg int64 remainder is zero");
    int64 seven = 7;
    check(seven / neg64 == 0 - 7, "minneg ordinary value by minus one");
}
`},

	{dir: "div_min_by_minus_one_wide", x64Only: true, src: `func main() {
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
	{dir: "alloc_basic", src: `import "std/mem";
func main() ! {
    *int p = mem.alloc(sizeof(int) * 4)!;
    p@ = 42;
    check(p@ == 42, "malloc write and read");
    mem.free(p);
    check(sizeof(int) == 8, "sizeof int");
    check(sizeof(int8) == 1, "sizeof int8");
    check(sizeof(int32) == 4, "sizeof int32");
}
`, expected: []string{"malloc write and read", "sizeof int", "sizeof int8", "sizeof int32"}},

	{dir: "alloc_reuse", src: `import "std/mem";
func main() ! {
    *int a = mem.alloc(24)!;
    mem.free(a);
    *int b = mem.alloc(24)!;
    check(a == b, "freed block is reused");
    mem.free(b);
    mem.free(null);
    check(true, "free null is a no-op");
}
`, expected: []string{"freed block is reused", "free null is a no-op"}},

	{dir: "alloc_realloc", src: `import "std/mem";
func main() ! {
    *int p = mem.alloc(8)!;
    p@ = 1234;
    *int same = mem.realloc(p, 8)!;
    check(same == p, "realloc within capacity keeps the block");
    *int big = mem.realloc(p, 100)!;
    check(big@ == 1234, "realloc keeps the contents");
    *int fresh = mem.realloc(null, 50)!;
    check(fresh != null, "realloc of null allocates");
    *int huge = mem.realloc(big, 100000)!;
    check(huge@ == 1234, "realloc to a large block keeps the contents");
    mem.free(huge);
    mem.free(fresh);
}
`, expected: []string{"realloc within capacity keeps the block", "realloc keeps the contents", "realloc of null allocates", "realloc to a large block keeps the contents"}},

	{dir: "abort_malloc_oom", exit: 4, src: `import "std/mem";
func main() ! {
    check(true, "oom before the failure");
    *int p = mem.alloc(-1)!;
    mem.free(p);
}
`, expected: []string{"oom before the failure"}, abort: "error[std/errors.Kind]: Kind.OutOfMemory"},

	{dir: "ptr_index", src: `import "std/mem";
struct P {
    int a;
    int8 b;
}

func main() ! {
    *int p = mem.alloc(sizeof(int) * 4)!;
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

    *int8 b = mem.alloc(16)!;
    b[4] = 7;
    *int8 q = (*int8)(b);
    check(q[4] == 7, "pointer index on bytes");
    int i = 4;
    check(b[i] == 7, "pointer index with a variable");

    *P ps = mem.alloc(sizeof(P) * 3)!;
    ps[2].a = 99;
    ps[2].b = 5;
    check(ps[2].a == 99, "pointer index struct field a");
    check(ps[2].b == 5, "pointer index struct field b");
    mem.free(ps);
    mem.free(b);
    mem.free(p);
}
`, expected: []string{"pointer index first", "pointer index second", "pointer index last", "pointer index compound assign", "pointer index through a converted pointer", "pointer index on bytes", "pointer index with a variable", "pointer index struct field a", "pointer index struct field b"}},

	{dir: "strings_core", src: `import "std/mem";
import "std/strings";

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
    mem.free((*char)(copy));
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

	{dir: "os_dir", src: `import "std/os";
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

	{dir: "os_mkdirall", src: `import "std/os";
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
    check(math.leadingZeros(1) == 63 && math.leadingZeros(0) == 64 && math.leadingZeros(0x8000000000000000) == 0 && math.leadingZeros(255) == 56, "leadingZeros");
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
    let ve38, e38 = math.checkedMul(math.MIN_INT, -1);
    check(e38 == errors.Kind.Overflow, "checkedMul of the minimum by -1 is Overflow");
    let ve40, e40 = math.checkedMul(-1, math.MIN_INT);
    check(e40 == errors.Kind.Overflow, "checkedMul of -1 by the minimum is Overflow");
    let ve42, e42 = math.checkedMul(3037000500, 3037000500);
    check(e42 == errors.Kind.Overflow, "checkedMul past the top is Overflow");
    let ve48, e48 = math.checkedAbs(math.MIN_INT);
    check(e48 == errors.Kind.Overflow, "checkedAbs of the minimum is Overflow");
    let ve52, e52 = math.pow(3, 40);
    check(e52 == errors.Kind.Overflow, "pow 3 to the 40 is Overflow");
}
`, expected: []string{"min and max of a single argument", "min and max of several arguments", "min and max of nothing are zero", "abs", "clamp", "mod is never negative", "gcd is not negative", "gcd with a zero", "sqrt rounds down", "sqrt of a negative is zero", "sqrt at the top of the range", "leadingZeros", "the signed limits", "the unsigned limits", "checkedAdd inside the range", "checkedSub inside the range", "checkedMul at the top of the range", "checkedMul of mixed signs", "checkedMul that lands on the minimum", "checkedMul by zero", "checkedAbs", "pow 2 to the 62", "pow 10 to the 18", "pow -2 to the 63 is the minimum", "pow with a zero", "lcm is not negative", "lcm with a zero", "checkedAdd past the top is Overflow", "checkedMul of the minimum by -1 is Overflow", "checkedMul of -1 by the minimum is Overflow", "checkedMul past the top is Overflow", "checkedAbs of the minimum is Overflow", "pow 3 to the 40 is Overflow"}},

	{dir: "math_core_wide", x64Only: true, src: `import "std/math";
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

	{dir: "target_files", files: map[string]string{"where/where.wsp": `import "std/math";

public func name() int {
    return which() + extra() + math.min(0, 1);
}

public func pieces() int {
    return both();
}
`, "where/where_x64.wsp": `import "std/math";

func which() int {
    return math.max(10, 3);
}

func both() int {
    return 100;
}
`, "where/where_linux_x64.wsp": `func extra() int {
    return 1;
}
`, "where/where_lua.wsp": `import "std/math";

func which() int {
    return math.max(10, 3);
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

	{dir: "ptr_slice", src: `import "std/mem";
func main() ! {
    *int8 b = mem.alloc(16)!;
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
    mem.free(b);
}
`, expected: []string{"pointer slice length", "pointer slice first", "pointer slice last", "pointer slice writes through to the buffer", "slice of a pointer slice", "pointer slice with variable bounds", "pointer slice with the start omitted begins at the pointer"}},

	{dir: "ptr_convert", x64Only: true, src: `import "std/mem";
struct P {
    int a;
    int b;
}

func main() ! {
    *int p = mem.alloc(32)!;
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
    mem.free(p);
}
`, expected: []string{"pointer to integer is not null", "pointer to int matches uint64", "integer to pointer round trip", "write through a converted integer", "read through a converted integer", "struct pointer reinterpreted from int pointer", "bytes view of a struct pointer", "converted zero is the null pointer"}},

	{dir: "ptr_arith", src: `import "std/mem";
struct P {
    int a;
    int8 b;
}

func main() ! {
    *int p = mem.alloc(sizeof(int) * 8)!;
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

    *int8 b = mem.alloc(16)!;
    *int8 c = b + 5;
    check(c - b == 5, "byte pointer difference");
    check(uint64(c) - uint64(b) == 5, "byte pointer address step");

    *P ps = mem.alloc(sizeof(P) * 4)!;
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
    mem.free(ps);
    mem.free(b);
    mem.free(p);
}
`, expected: []string{"pointer plus offset", "pointer difference", "offset plus pointer", "pointer ordering", "pointer ordering reversed", "pointer minus assign", "pointer plus assign", "pointer minus offset", "byte pointer difference", "byte pointer address step", "struct pointer difference", "struct pointer steps by sizeof", "pointer plus variable", "pointer plus narrow integer", "pointer increment steps one element", "pointer decrement steps one element", "byte pointer increment", "struct pointer increment"}},

	{dir: "string_from_bytes", src: `import "std/mem";
func main() ! {
    *int8 b = mem.alloc(8)!;
    b[0] = 104;
    b[1] = 105;
    string s = string(b[0:2]);
    check(len(s) == 2, "string from int8 pointer slice length");
    check(s == "hi", "string from int8 pointer slice content");

    *char c = mem.alloc(4)!;
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
    mem.free(b);
    mem.free(c);
}
`, expected: []string{"string from int8 pointer slice length", "string from int8 pointer slice content", "string from char slice", "string from uint8 array slice", "string from inline slice", "string from bytes length"}},

	{dir: "free_slice", src: `import "std/mem";
func main() ! {
    *int8 b = mem.alloc(24)!;
    b[0] = 120;
    int8[] s = b[0:5];
    mem.free((*int8)(s));
    *int8 c = mem.alloc(24)!;
    check(b == c, "free of a slice releases its block");
    mem.free(c);
    mem.free(null);
    check(true, "free of null still works");
}
`, expected: []string{"free of a slice releases its block", "free of null still works"}},

	{dir: "free_many", x64Only: true, src: `import "std/mem";
func main() ! {
    *int a = mem.alloc(24)!;
    *int b = mem.alloc(24)!;
    *int c = mem.alloc(100)!;
    mem.free(a, b, c);
    *int x = mem.alloc(24)!;
    *int y = mem.alloc(24)!;
    *int z = mem.alloc(100)!;
    check((x == a && y == b) || (x == b && y == a), "free of many releases every small block");
    check(z == c, "free of many releases the last block");

    *int8 raw = mem.alloc(40)!;
    int8[] s = raw[0:3];
    *int d = mem.alloc(40)!;
    mem.free((*int8)(s), d);
    *int8 raw2 = mem.alloc(40)!;
    *int d2 = mem.alloc(40)!;
    check((raw2 == raw && uint64(d2) == uint64(d)) || (uint64(raw2) == uint64(d) && d2 == (*int)(raw)), "free of mixed slice and pointer");
    mem.free(raw2, d2, null);
    check(true, "free of many accepts null");
}
`, expected: []string{"free of many releases every small block", "free of many releases the last block", "free of mixed slice and pointer", "free of many accepts null"}},

	{dir: "free_node_next", src: `import "std/mem";
struct Node {
    *Node next;
    int v;
}

func main() ! {
    *Node first = mem.alloc(sizeof(Node))!;
    *Node second = mem.alloc(sizeof(Node))!;
    first.next = second;
    second.v = 42;
    second.next = null;
    mem.free(first, first.next);
    *Node a = mem.alloc(sizeof(Node))!;
    *Node b = mem.alloc(sizeof(Node))!;
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

	{dir: "fmt_core", src: `import "std/fmt";
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
`, expected: []string{"decimal", "negative decimal", "zero", "int8", "d type letter", "hex lower case", "hex upper case", "hex prefix", "binary", "binary prefix", "octal", "octal prefix", "negative hex", "number aligns right", "left align", "center with even padding", "center with odd padding puts the extra space on the right", "zero padding", "zero padding goes after the sign", "zero padded hex", "zeros go after the prefix", "width smaller than the text changes nothing", "custom fill", "custom fill with center", "string", "string aligns left", "string right align", "precision cuts a string", "width and precision", "empty string", "char", "char as a number", "char in hex", "integer as a char", "true", "false", "bool aligns left", "an enum prints its number", "an enum by name through name()", "plain text", "text around a placeholder", "several placeholders of different types", "escaped braces", "an escaped brace before a placeholder", "an escaped brace after a placeholder", "fewer arguments keep the placeholder as written", "no arguments", "a type letter that does not fit the type is written back and reported as WrongType", "a number with no digits in a spec is BadSpec", "an unknown type letter is BadSpec", "text after the type letter is BadSpec", "an open brace with no close is Unmatched", "a lone close brace is Unmatched", "an extra argument is ExtraArgument", "an argument of a named type is UnsupportedType", "a problem does not stop the text after it, and takes its argument", "fewer arguments are not an error", "a good format reports no error", "sprintf returns the byte count", "sprintf into a buffer that is too small is NoSpace", "a memory writer collects the text", "a second call appends", "a memory writer that fills up is NoSpace", "a full writer holds the text until flush", "flush writes the text", "a line writer holds the text until a newline", "a line writer flushes at the newline", "a writer with no buffering writes at once", "a small buffer is flushed as it fills", "fprintf writes everything in one call", "fprintf still writes the text when the format has a problem"}},

	{dir: "fmt_core_wide", x64Only: true, src: `import "std/fmt";
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
`, expected: []string{"decimal", "negative decimal", "zero", "largest uint", "int8", "d type letter", "hex lower case", "hex upper case", "hex prefix", "binary", "binary prefix", "octal", "octal prefix", "negative hex", "number aligns right", "left align", "center with even padding", "center with odd padding puts the extra space on the right", "zero padding", "zero padding goes after the sign", "zero padded hex", "zeros go after the prefix", "width smaller than the text changes nothing", "custom fill", "custom fill with center", "string", "string aligns left", "string right align", "precision cuts a string", "width and precision", "empty string", "char", "char as a number", "char in hex", "integer as a char", "true", "false", "bool aligns left", "an enum prints its number", "an enum by name through name()", "plain text", "text around a placeholder", "several placeholders of different types", "escaped braces", "an escaped brace before a placeholder", "an escaped brace after a placeholder", "fewer arguments keep the placeholder as written", "no arguments", "a type letter that does not fit the type is written back and reported as WrongType", "a number with no digits in a spec is BadSpec", "an unknown type letter is BadSpec", "text after the type letter is BadSpec", "an open brace with no close is Unmatched", "a lone close brace is Unmatched", "an extra argument is ExtraArgument", "an argument of a named type is UnsupportedType", "a problem does not stop the text after it, and takes its argument", "fewer arguments are not an error", "a good format reports no error", "sprintf returns the byte count", "sprintf into a buffer that is too small is NoSpace", "a memory writer collects the text", "a second call appends", "a memory writer that fills up is NoSpace", "a full writer holds the text until flush", "flush writes the text", "a line writer holds the text until a newline", "a line writer flushes at the newline", "a writer with no buffering writes at once", "a small buffer is flushed as it fills", "fprintf writes everything in one call", "fprintf still writes the text when the format has a problem"}},

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

	{dir: "any_variadic_float", src: `func kind(...args) int {
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
    float32 small = 2.5;
    check(kind(1.5) == 99, "case float matches a float");
    check(kind(2) == 1, "an int is not a float");
    check(kind(small) == -1, "a float32 is not a float");
}
`, expected: []string{"case float matches a float", "an int is not a float", "a float32 is not a float"}},

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
`}, src: `import "std/mem";
extern func add(int32 a, int32 b) int32;
extern func mul64(int64 a, int64 b) int64;
extern func sum_bytes(*int8 p, int32 n) int32;
extern func twice(int32 x) int32 as c_twice;
extern func low_byte(uint32 v) uint8;
extern func count_calls() int32;

func main() ! {
    check(add(2, 3) == 5, "extern function with two arguments");
    check(mul64(6, 7) == 42, "extern function with 64-bit arguments");
    *int8 b = mem.alloc(4)!;
    b[0] = 1;
    b[1] = 2;
    b[2] = 3;
    b[3] = 4;
    check(sum_bytes(b, 4) == 10, "extern function taking a pointer");
    mem.free(b);
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

	{dir: "interop_libc_float", x64Only: true, libc: true, src: `extern func printf(*int8 fmt, ...) int32;

func main() int {
    float32 narrow = 1.5;
    printf((*int8)("%f %d %f\n"), 2.5, 7, 4.25);
    printf((*int8)("%f\n"), narrow);
    printf((*int8)("%f %f %f %f %f %f %f %f %f %f\n"), 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0);
    printf((*int8)("no floats\n"));
    return 0;
}
`, expected: []string{"2.500000 7 4.250000", "1.500000", "1.000000 2.000000 3.000000 4.000000 5.000000 6.000000 7.000000 8.000000 9.000000 10.000000", "no floats"}},

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
	{dir: "libm_exp", src: `import "std/math";

float[26] expIn = [-511.8900582426383, 486.40723171212585, 369.28446656725964, -342.9033639648096, -6.390878071282714, -70.71250929576661, 212.23016181186813, 404.21269158971836, -568.5965785160712, -660.3135328691911, 470.0711454878176, -94.12610493292527, 0.9120685437784988, 0.8956549741186988, -0.8868972645463826, -0.8302560096821567, 0.6709977562588991, 0.4719399781370466, 0.3394608028804418, -0.3837270848217116, 0.0, 1.0, -1.0, 709.0, -745.0, 1e-20];
float[26] expWant = [4.886212654088617e-223, 1.753786369552851e+211, 2.388944623308051e+160, 1.1993921554110276e-149, 0.0016767832195999158, 1.9496085075269736e-31, 1.480430992329736e+92, 3.526480385993986e+175, 1.1525068661192175e-247, 1.6961972269773522e-287, 1.41027753313563e+204, 1.322976179199616e-41, 2.4894667820224923, 2.4489392561214323, 0.4119318876463562, 0.43593766777066423, 1.9561881462163546, 1.6031011597207216, 1.4041902510656181, 0.6813173436529935, 1.0, 2.718281828459045, 0.36787944117144233, 8.218407461554972e+307, 5e-324, 1.0];
float[26] expm1In = [-15.722122374486517, 2.65375351775711, -7.802690007115245, 6.235202315771666, 7.543218246483242, -26.068268445611213, -29.20992050670755, 20.2481449257876, -14.438759140319542, -15.940142337198218, 29.738690130627766, -1.7841895486531243, -0.527903820525131, -0.7936679315385684, -0.20788351477863798, -0.6900554583951795, -0.8669698086408202, -0.19681797102985032, 0.8359100861754378, 0.600904702991617, 1e-10, -1e-10, 0.25, -0.25, 40.0, -50.0];
float[26] expm1Want = [-0.999999851417039, 13.207265913020407, -0.9995913657299079, 509.40387075612756, 1886.8959736464433, -0.999999999995228, -0.9999999999997938, 621809867.1678388, -0.9999994638002837, -0.9999998805230459, 8229037146411.302, -0.8320668925236464, -0.41015991460554746, -0.5478168271343251, -0.18769834766104365, -0.49845174676647747, -0.5797770220246046, -0.1786598725963487, 1.3069125824679428, 0.8237680226359645, 1.00000000005e-10, -9.999999999500001e-11, 0.2840254166877415, -0.22119921692859512, 2.3538526683701997e+17, -1.0];
float[27] exp2In = [245.80338977940391, 483.5739785214587, 590.3871311313933, 884.9005675541007, 479.79714947986145, 844.6499933308339, -941.9895434327706, -68.75469124378935, 886.7134339966274, 297.94910627384843, 1.7600405025699777, 1.931724253918361, -0.08979223241432788, -1.4302711023320525, -2.997289710689574, 0.9769113773026055, -0.17847445761332992, 1.5583838105873582, -0.7610377675569504, 1.6208390156279409, 0.0, 1.0, -1.0, 10.0, 0.5, -1074.0, 1023.5];
float[27] exp2Want = [9.867187013299923e+73, 3.7176855852651323e+145, 5.2995076639766425e+177, 2.407764538388296e+266, 2.712275665605046e+144, 1.8407034654891507e+254, 2.709516655103489e-284, 2.008054949959655e-21, 8.459431755429353e+266, 4.916071262359412e+89, 3.3870763377167425, 3.8151089407358665, 0.9396580629319634, 0.37106115858678024, 0.12523504939047095, 1.9682471337415564, 0.883636882533748, 2.9452371628460092, 0.5900717242876609, 3.075538456777682, 1.0, 2.0, 0.5, 1024.0, 1.4142135623730951, 5e-324, 1.2711610061536464e+308];

func ulps(float a, float b) float {
    if math.isNaN(b) {
        if math.isNaN(a) {
            return 0.0;
        }
        return 1000000000.0;
    }
    if math.isNaN(a) {
        return 1000000000.0;
    }
    if math.isInf(b, 0) || math.isInf(a, 0) {
        if a == b {
            return 0.0;
        }
        return 1000000000.0;
    }
    if b == 0.0 {
        if a == 0.0 && math.signbit(a) == math.signbit(b) {
            return 0.0;
        }
        return 1000000000.0;
    }
    let _, e = math.frexp(b);
    float ulp = math.ldexp(1.0, e - 53);
    if ulp < 5e-324 {
        ulp = 5e-324;
    }
    return math.fabs(a - b) / ulp;
}

func main() {
    float expWorst = 0.0;
    for i in 0..25 {
        float u = ulps(math.exp(expIn[i]), expWant[i]);
        if u > expWorst {
            expWorst = u;
        }
    }
    check(expWorst <= 1.0, "exp stays within 1 ulp");
    check(ulps(math.exp(math.inf(1)), math.inf(1)) == 0.0 && ulps(math.exp(math.inf(-1)), 0.0) == 0.0 && ulps(math.exp(math.nan()), math.nan()) == 0.0 && ulps(math.exp(710.0), math.inf(1)) == 0.0 && ulps(math.exp(-746.0), 0.0) == 0.0 && ulps(math.exp(0.0), 1.0) == 0.0, "exp special values");
    float expm1Worst = 0.0;
    for i in 0..25 {
        float u = ulps(math.expm1(expm1In[i]), expm1Want[i]);
        if u > expm1Worst {
            expm1Worst = u;
        }
    }
    check(expm1Worst <= 1.0, "expm1 stays within 1 ulp");
    check(ulps(math.expm1(math.inf(1)), math.inf(1)) == 0.0 && ulps(math.expm1(math.inf(-1)), -1.0) == 0.0 && ulps(math.expm1(math.nan()), math.nan()) == 0.0 && ulps(math.expm1(0.0), 0.0) == 0.0, "expm1 special values");
    float exp2Worst = 0.0;
    for i in 0..26 {
        float u = ulps(math.exp2(exp2In[i]), exp2Want[i]);
        if u > exp2Worst {
            exp2Worst = u;
        }
    }
    check(exp2Worst <= 2.0, "exp2 stays within 2 ulp");
    check(ulps(math.exp2(math.inf(1)), math.inf(1)) == 0.0 && ulps(math.exp2(math.inf(-1)), 0.0) == 0.0 && ulps(math.exp2(math.nan()), math.nan()) == 0.0 && ulps(math.exp2(1024.0), math.inf(1)) == 0.0 && ulps(math.exp2(0.0), 1.0) == 0.0, "exp2 special values");
}
`, expected: []string{"exp stays within 1 ulp", "exp special values", "expm1 stays within 1 ulp", "expm1 special values", "exp2 stays within 2 ulp", "exp2 special values"}},

	{dir: "libm_log", src: `import "std/math";

float[27] logIn = [2.6775161921773473e-29, 7.30182883550128e+35, 3.360086163397612e+254, 2.454946656957562e-21, 50671.50894967005, 2.6971016203361873e+52, 6.254697535355299e-190, 13969582.443943003, 8.504173854606704e+77, 6.1111579958679464e+175, 2.979022185553462e-244, 1.0983925402356037e-118, 1.2118560180328222, 1.486208753985883, 1.499615706687257, 0.7139005293880516, 0.516290664635102, 1.0621317380950466, 0.9110722092174971, 1.7155220783526257, 1.0, 2.0, 0.5, 10.0, 1e-310, 5e-324, 1.7976931348623157e+308];
float[27] logWant = [-65.7900781260073, 82.57860309698083, 586.0685802380023, -47.4561819201914, 10.833119077968718, 120.72660255816527, -435.6578348817776, 16.452392841304913, 179.4396092452347, 404.76250755387593, -560.739167569528, -271.61119318933135, 0.19215308358757793, 0.39622841690413174, 0.40520887974265163, -0.3370116409302913, -0.6610853685400678, 0.060277962297010046, -0.09313312117787143, 0.539717453156387, 0.0, 0.6931471805599453, -0.6931471805599453, 2.302585092994046, -713.8013788281542, -744.4400719213812, 709.782712893384];
float[26] log2In = [2.5415287642773857e-145, 1.4282221978440683e+111, 2.812877438393363e+110, 3.996656595088988e+209, 2.7195955647961264e-189, 2.1635422434790114e-162, 1.9767461474140362e-212, 1.2524527113607414e-165, 2.5951432026823167e+140, 1.3421889401951906e-222, 6.149660040717195e+18, 2.2106413795880964e-172, 0.6602428065524835, 1.553878285980283, 1.478063030471413, 1.9105285843491768, 0.9066728398404847, 0.8836632701495588, 1.601089046217045, 1.4876750273601136, 1.0, 2.0, 0.5, 8.0, 1e-310, 1024.0];
float[26] log2Want = [-480.3338771997893, 369.24823897880424, 366.90413713108217, 696.2817654487891, -626.4010178116645, -537.038956081886, -703.2656285032133, -547.7933795235518, 466.44574743441285, -737.0434492907023, 62.41521236702153, -570.2271673171156, -0.5989314167950222, 0.6358735030444475, 0.5637077930359022, 0.9339718427204481, -0.14134602647686306, -0.17843137576057702, 0.6790535470567153, 0.5730594137985336, 0.0, 1.0, -1.0, 3.0, -1029.7977094150824, 10.0];
float[25] log10In = [1.397284788912151e+279, 9.835949159798784e-294, 3.935245927290424e+141, 6.421475927747785e-206, 6.363132954435529e+291, 1.3439791298883743e-290, 4.951774748296471e+227, 6.46247150022087e+108, 2.5434868047505643e+214, 7.75783430764046e+299, 6.746978065682069e-157, 7.025129818036019e-98, 1.0422841623711128, 1.220720998401941, 1.1254289399084971, 1.1701388806758848, 1.1142735949248443, 1.4865587395914062, 0.8882830728351174, 1.4524938861552261, 1.0, 10.0, 100.0, 1e-05, 1e+22];
float[25] log10Want = [279.1452849312738, -293.00718372471533, 141.59497187812528, -205.1923651411511, 291.8036709976939, -289.87160747521426, 227.69476088080268, 108.81039864084875, 214.40542948891493, 299.88974049967163, -156.17089070172685, -97.15334564598007, 0.01798613866148298, 0.0866164152141013, 0.05131807865249468, 0.06823741006386931, 0.04699183911196543, 0.1721820745072574, -0.05144861374868377, 0.16211431303163742, 0.0, 1.0, 2.0, -5.0, 22.0];
float[27] log1pIn = [2.136683619177834, 3.8420777549408873, 4.763363698200021, 0.7448560128091484, 3.598983552409736, 3.2282770023920415, 2.971684512770744, -0.33012932698590114, -0.8286486950447445, 1.311184916220342, 3.480907225036643, 0.5217787674041998, -0.0006374702733335574, 0.0003228610969904888, -0.00033088059366499995, -0.0006041207916749592, -2.096932190376907e-05, -1.1930641785649421e-05, -4.0454919703713675e-05, -8.333777059837976e-05, 1e-20, -1e-20, 1e-10, -0.5, 0.3, 3.0, 1e+16];
float[27] log1pWant = [1.1431660695408756, 1.5773439168216459, 1.7515212796424795, 0.5566720381004255, 1.5258353122108255, 1.44179458203788, 1.3791903152264438, -0.4006706105810075, -1.7640394149609855, 0.8377603437641773, 1.4998255315139413, 0.41987989237237416, -0.0006376735438988018, 0.00032280898856206836, -0.00033093534672678213, -0.0006043033461674385, -2.0969541763073163e-05, -1.1930712956322205e-05, -4.0455738026047974e-05, -8.334124338332803e-05, 1e-20, -1e-20, 9.999999999500001e-11, -0.6931471805599453, 0.26236426446749106, 1.3862943611198906, 36.841361487904734];

func ulps(float a, float b) float {
    if math.isNaN(b) {
        if math.isNaN(a) {
            return 0.0;
        }
        return 1000000000.0;
    }
    if math.isNaN(a) {
        return 1000000000.0;
    }
    if math.isInf(b, 0) || math.isInf(a, 0) {
        if a == b {
            return 0.0;
        }
        return 1000000000.0;
    }
    if b == 0.0 {
        if a == 0.0 && math.signbit(a) == math.signbit(b) {
            return 0.0;
        }
        return 1000000000.0;
    }
    let _, e = math.frexp(b);
    float ulp = math.ldexp(1.0, e - 53);
    if ulp < 5e-324 {
        ulp = 5e-324;
    }
    return math.fabs(a - b) / ulp;
}

func main() {
    float logWorst = 0.0;
    for i in 0..26 {
        float u = ulps(math.log(logIn[i]), logWant[i]);
        if u > logWorst {
            logWorst = u;
        }
    }
    check(logWorst <= 1.0, "log stays within 1 ulp");
    check(ulps(math.log(0.0), math.inf(-1)) == 0.0 && ulps(math.log(-1.0), math.nan()) == 0.0 && ulps(math.log(math.inf(1)), math.inf(1)) == 0.0 && ulps(math.log(math.nan()), math.nan()) == 0.0 && ulps(math.log(1.0), 0.0) == 0.0, "log special values");
    float log2Worst = 0.0;
    for i in 0..25 {
        float u = ulps(math.log2(log2In[i]), log2Want[i]);
        if u > log2Worst {
            log2Worst = u;
        }
    }
    check(log2Worst <= 2.0, "log2 stays within 2 ulp");
    check(ulps(math.log2(0.0), math.inf(-1)) == 0.0 && ulps(math.log2(-1.0), math.nan()) == 0.0 && ulps(math.log2(math.inf(1)), math.inf(1)) == 0.0 && ulps(math.log2(math.nan()), math.nan()) == 0.0 && ulps(math.log2(8.0), 3.0) == 0.0, "log2 special values");
    float log10Worst = 0.0;
    for i in 0..24 {
        float u = ulps(math.log10(log10In[i]), log10Want[i]);
        if u > log10Worst {
            log10Worst = u;
        }
    }
    check(log10Worst <= 2.0, "log10 stays within 2 ulp");
    check(ulps(math.log10(0.0), math.inf(-1)) == 0.0 && ulps(math.log10(-1.0), math.nan()) == 0.0 && ulps(math.log10(math.inf(1)), math.inf(1)) == 0.0 && ulps(math.log10(math.nan()), math.nan()) == 0.0 && ulps(math.log10(1000.0), 3.0) == 0.0, "log10 special values");
    float log1pWorst = 0.0;
    for i in 0..26 {
        float u = ulps(math.log1p(log1pIn[i]), log1pWant[i]);
        if u > log1pWorst {
            log1pWorst = u;
        }
    }
    check(log1pWorst <= 1.0, "log1p stays within 1 ulp");
    check(ulps(math.log1p(-1.0), math.inf(-1)) == 0.0 && ulps(math.log1p(-2.0), math.nan()) == 0.0 && ulps(math.log1p(math.inf(1)), math.inf(1)) == 0.0 && ulps(math.log1p(math.nan()), math.nan()) == 0.0 && ulps(math.log1p(0.0), 0.0) == 0.0, "log1p special values");
}
`, expected: []string{"log stays within 1 ulp", "log special values", "log2 stays within 2 ulp", "log2 special values", "log10 stays within 2 ulp", "log10 special values", "log1p stays within 1 ulp", "log1p special values"}},

	{dir: "libm_pow", src: `import "std/math";

float[39] fpowIn = [8.255824496861566, 31.753647020831988, 10.808557251216204, 40.394636554651846, 25.259156831965885, 0.16899972420875523, 29.27227123413439, 39.68961358262968, 11.642420620463687, 49.878084831377095, 43.75322565029849, 1.713162266155488, 24.882640380850667, 47.59868621502933, 1.0836427448893351e-60, 6.830901365858836e-66, 7.201601543448497e-09, 5.160080461344857e-28, 4.078322563754017e-55, 3.6377771570065726e+63, 4.85328359299931e+48, 2.7134108841713474e+81, -2.527554730339845, -0.2106904139772201, -8.78746907826351, -4.466468335781403, -8.009313709451346, -1.2131588123392447, -7.109325777622437, -5.035752935564525, 2.0, 2.0, 10.0, 1.5, -2.0, 3.0, 2.0, 2.0, 1.0000000001];
float[39] fpowIn2 = [7.590676968702695, -0.8359811177019516, 11.705334328009855, 0.498249585503725, -10.55774546015158, -5.158132334715688, -17.229409472667548, -10.709891342671213, -18.306583333952453, 9.547510078758435, 4.635603559565624, -6.842826145842867, -15.367607578988022, -5.262134366266684, 1.9050566593615956, 2.5676631787217934, 0.31201724984309376, -0.09177816728811372, -2.77716267988345, -1.5908032771368326, -1.748626708665743, 2.363794982045693, 9.0, -7.0, 6.0, 3.0, -5.0, 4.0, 5.0, 1.0, 10.0, 0.5, -3.0, 2.5, 3.0, 200.0, -1074.0, 1023.0, 1000000000.0];
float[39] fpowWant = [9095622.966986442, 0.0555301564475088, 1260656784507.3506, 6.314662157316893, 1.5617760528415504e-15, 9608.73488029489, 5.417786106616778e-26, 7.556913017140847e-18, 3.0506315374422734e-20, 1.6248401325856884e+16, 40464311.11753076, 0.025127976108645003, 3.535175160270144e-22, 1.486859314289955e-09, 5.795094043904555e-115, 4.75213396868653e-168, 0.002879900064769065, 319.4349114600753, 1.1182547580017725e+151, 7.712828027727136e-102, 7.350175174883073e-86, 3.105528058989683e+192, -4210.2231523713, -54260.846503486966, 460450.40457081364, -89.10309272546003, -3.034055198348414e-05, 2.1660607312140896, -18161.097037790903, -5.035752935564525, 1024.0, 1.4142135623730951, 0.001, 2.7556759606310752, -8.0, 2.6561398887587478e+95, 5e-324, 8.98846567431158e+307, 1.105170927214347];

func ulps(float a, float b) float {
    if math.isNaN(b) {
        if math.isNaN(a) {
            return 0.0;
        }
        return 1000000000.0;
    }
    if math.isNaN(a) {
        return 1000000000.0;
    }
    if math.isInf(b, 0) || math.isInf(a, 0) {
        if a == b {
            return 0.0;
        }
        return 1000000000.0;
    }
    if b == 0.0 {
        if a == 0.0 && math.signbit(a) == math.signbit(b) {
            return 0.0;
        }
        return 1000000000.0;
    }
    let _, e = math.frexp(b);
    float ulp = math.ldexp(1.0, e - 53);
    if ulp < 5e-324 {
        ulp = 5e-324;
    }
    return math.fabs(a - b) / ulp;
}

func main() {
    float fpowWorst = 0.0;
    for i in 0..38 {
        float u = ulps(math.fpow(fpowIn[i], fpowIn2[i]), fpowWant[i]);
        if u > fpowWorst {
            fpowWorst = u;
        }
    }
    check(fpowWorst <= 1.0, "fpow stays within 1 ulp");
    check(ulps(math.fpow(2.0, 0.0), 1.0) == 0.0 && ulps(math.fpow(math.nan(), 0.0), 1.0) == 0.0 && ulps(math.fpow(1.0, math.nan()), 1.0) == 0.0 && ulps(math.fpow(math.nan(), 2.0), math.nan()) == 0.0 && ulps(math.fpow(-8.0, 0.3333333333333333), math.nan()) == 0.0 && ulps(math.fpow(0.0, -1.0), math.inf(1)) == 0.0 && ulps(math.fpow((0.0 - 0.0) * -1.0, -1.0), math.inf(-1)) == 0.0 && ulps(math.fpow(0.0, 3.0), 0.0) == 0.0 && ulps(math.fpow(2.0, 1024.0), math.inf(1)) == 0.0 && ulps(math.fpow(2.0, -1075.0), 0.0) == 0.0 && ulps(math.fpow(math.inf(-1), 3.0), math.inf(-1)) == 0.0 && ulps(math.fpow(math.inf(1), -1.0), 0.0) == 0.0 && ulps(math.fpow(0.5, math.inf(1)), 0.0) == 0.0 && ulps(math.fpow(2.0, math.inf(-1)), 0.0) == 0.0 && ulps(math.fpow(-1.0, math.inf(1)), 1.0) == 0.0 && ulps(math.fpow(-1.0, 3.0), -1.0) == 0.0 && ulps(math.fpow(4.0, 0.5), 2.0) == 0.0 && ulps(math.fpow(10.0, 22.0), 1e+22) == 0.0, "fpow special values");
}
`, expected: []string{"fpow stays within 1 ulp", "fpow special values"}},

	{dir: "libm_trig", src: `import "std/math";

float[48] sinIn = [-0.7312715117751976, 0.6948674738744653, 0.5275492379532281, -0.4898619485211566, -0.009129825816118098, -0.10101787042252375, 0.3031859454455259, 0.5774467022710263, 3.648274175113995, 3.582619896474795, -3.5475890581855305, -3.321024038728627, 2.6839910250355965, 1.8877599125481863, 1.3578432115217671, -1.5349083392868463, -52.40707458162173, 8.845845059190367, -26.008966690384156, 20.7840077192389, 25.144060821610807, -86.89422815203739, -97.36640168902517, 67.49381641929199, -527903.820525131, -793667.9315385683, -207883.51477863803, -690055.4583951795, -866969.8086408202, -196817.9710298503, 9256080398171218.0, 7.389104833201065e+17, 5.285633533007853e+18, 1.200064366491094e+21, 6.892506682481676e+17, 5.7174180267874065e+20, -2.0665991623833704e+299, -1.780459576802733e+299, -5.149653720690547e+299, 0.5, 3.141592653589793, 1.5707963267948966, 6.283185307179586, 1e-10, 1e+300, 1.7976931348623157e+308, 1e+22, 2000000.0];
float[48] sinWant = [-0.6678165936974793, 0.6402836467265434, 0.5034172903243089, -0.470504076330938, -0.00912969898249011, -0.10084615006535798, 0.29856235164836836, 0.5458864145547881, -0.48527838036048504, -0.4268686398796877, 0.3949343624764782, 0.17847011636901, 0.44179776546647814, 0.9501861951184719, 0.9774110448580267, -0.9993560952903914, -0.8414712708463894, 0.5471310369711966, -0.7683284427854311, 0.9346028453968994, 0.011319351158098536, 0.8773762422250856, -0.022968552252065972, -0.9987288971489507, 0.015793206257686192, -0.889512390595496, 0.9273613350533669, 0.9967732899104907, 0.19081556755929413, 0.3267616548527848, 0.6896795585620723, 0.5498580618180777, 0.2501930087917851, -0.07468053365477546, 0.9064321487056509, 0.9367572192861974, -0.9559485715868026, -0.13692738568869572, 0.9918382307080797, 0.479425538604203, 1.2246467991473532e-16, 1.0, -2.4492935982947064e-16, 1e-10, -0.8178819121159085, 0.004961954789184062, -0.8522008497671888, -0.65571431556347];
float[48] cosIn = [-0.7312715117751976, 0.6948674738744653, 0.5275492379532281, -0.4898619485211566, -0.009129825816118098, -0.10101787042252375, 0.3031859454455259, 0.5774467022710263, 3.648274175113995, 3.582619896474795, -3.5475890581855305, -3.321024038728627, 2.6839910250355965, 1.8877599125481863, 1.3578432115217671, -1.5349083392868463, -52.40707458162173, 8.845845059190367, -26.008966690384156, 20.7840077192389, 25.144060821610807, -86.89422815203739, -97.36640168902517, 67.49381641929199, -527903.820525131, -793667.9315385683, -207883.51477863803, -690055.4583951795, -866969.8086408202, -196817.9710298503, 9256080398171218.0, 7.389104833201065e+17, 5.285633533007853e+18, 1.200064366491094e+21, 6.892506682481676e+17, 5.7174180267874065e+20, -2.0665991623833704e+299, -1.780459576802733e+299, -5.149653720690547e+299, 0.5, 3.141592653589793, 1.5707963267948966, 6.283185307179586, 1e-10, 1e+300, 1.7976931348623157e+308, 1e+22, 2000000.0];
float[48] cosWant = [0.7443258676025548, 0.7681385628482396, 0.8640434200909873, 0.8823978208019163, 0.9999583234297763, 0.9949020323715272, 0.9543901310146685, 0.8378591900817929, -0.8743597049010804, -0.9043136426524071, -0.9187093388745419, -0.9839453326090998, -0.8971146718390167, -0.31168284297709, 0.21134722470271688, 0.03588028436261607, -0.5403018603893219, -0.8370469690422534, 0.6400557819494443, -0.3556930156413243, 0.9999359340924596, 0.47980301122334346, -0.9997361880053378, -0.05040426568893402, -0.9998752795404546, 0.456911049305098, -0.3741670138427985, -0.08026835317244695, -0.9816259059219171, -0.9451067775219208, 0.7241147053482792, 0.8352581109176175, -0.9681959813755239, -0.9972075099461686, 0.4223514647693988, -0.3499798738716144, -0.2935342032560292, 0.9905810875685338, 0.12750264352501098, 0.8775825618903728, -1.0, 6.123233995736766e-17, 1.0, 1.0, -0.5753861119575491, -0.9999876894265599, 0.523214785395139, 0.7550090968757464];
float[47] tanIn = [-0.7312715117751976, 0.6948674738744653, 0.5275492379532281, -0.4898619485211566, -0.009129825816118098, -0.10101787042252375, 0.3031859454455259, 0.5774467022710263, 3.648274175113995, 3.582619896474795, -3.5475890581855305, -3.321024038728627, 2.6839910250355965, 1.8877599125481863, 1.3578432115217671, -1.5349083392868463, -52.40707458162173, 8.845845059190367, -26.008966690384156, 20.7840077192389, 25.144060821610807, -86.89422815203739, -97.36640168902517, 67.49381641929199, -527903.820525131, -793667.9315385683, -207883.51477863803, -690055.4583951795, -866969.8086408202, -196817.9710298503, 9256080398171218.0, 7.389104833201065e+17, 5.285633533007853e+18, 1.200064366491094e+21, 6.892506682481676e+17, 5.7174180267874065e+20, -2.0665991623833704e+299, -1.780459576802733e+299, -5.149653720690547e+299, 0.5, 3.141592653589793, 6.283185307179586, 1e-10, 1e+300, 1.7976931348623157e+308, 1e+22, 2000000.0];
float[47] tanWant = [-0.8972099758516939, 0.8335522751941874, 0.5826296209411526, -0.533210832165641, -0.009130079492889244, -0.10136289482188826, 0.31283051023479164, 0.651525245550506, 0.5550100006214106, 0.4720360500452652, -0.4298795557692813, -0.18138214640011135, -0.49246521022872847, -3.04856753115639, 4.624669409465218, -27.852513240715226, 1.5574095381423556, -0.6536443678868132, -1.2004085650867826, -2.627554672986156, 0.01132007638906583, 1.8286176236953124, 0.022974613230608933, 19.814372523796457, -0.015795176239325363, -1.9467955348165207, -2.478468974400255, -12.41801096590387, -0.1943872573127389, -0.3457404630083778, 0.952445176804351, 0.6583091557339104, -0.2584115340329484, 0.07488966229186027, 2.1461560437598033, -2.6766031112687205, 3.256685459421558, -0.13822935588725577, 7.778962092762573, 0.5463024898437905, -1.2246467991473532e-16, -2.4492935982947064e-16, 1e-10, 1.4214488238747245, -0.004962015874444895, -1.6287782256068988, -0.8684853179608544];

func ulps(float a, float b) float {
    if math.isNaN(b) {
        if math.isNaN(a) {
            return 0.0;
        }
        return 1000000000.0;
    }
    if math.isNaN(a) {
        return 1000000000.0;
    }
    if math.isInf(b, 0) || math.isInf(a, 0) {
        if a == b {
            return 0.0;
        }
        return 1000000000.0;
    }
    if b == 0.0 {
        if a == 0.0 && math.signbit(a) == math.signbit(b) {
            return 0.0;
        }
        return 1000000000.0;
    }
    let _, e = math.frexp(b);
    float ulp = math.ldexp(1.0, e - 53);
    if ulp < 5e-324 {
        ulp = 5e-324;
    }
    return math.fabs(a - b) / ulp;
}

func main() {
    float sinWorst = 0.0;
    for i in 0..47 {
        float u = ulps(math.sin(sinIn[i]), sinWant[i]);
        if u > sinWorst {
            sinWorst = u;
        }
    }
    check(sinWorst <= 1.0, "sin stays within 1 ulp");
    check(ulps(math.sin(math.inf(1)), math.nan()) == 0.0 && ulps(math.sin(math.inf(-1)), math.nan()) == 0.0 && ulps(math.sin(math.nan()), math.nan()) == 0.0 && ulps(math.sin(0.0), 0.0) == 0.0, "sin special values");
    float cosWorst = 0.0;
    for i in 0..47 {
        float u = ulps(math.cos(cosIn[i]), cosWant[i]);
        if u > cosWorst {
            cosWorst = u;
        }
    }
    check(cosWorst <= 1.0, "cos stays within 1 ulp");
    check(ulps(math.cos(math.inf(1)), math.nan()) == 0.0 && ulps(math.cos(math.nan()), math.nan()) == 0.0 && ulps(math.cos(0.0), 1.0) == 0.0, "cos special values");
    float tanWorst = 0.0;
    for i in 0..46 {
        float u = ulps(math.tan(tanIn[i]), tanWant[i]);
        if u > tanWorst {
            tanWorst = u;
        }
    }
    check(tanWorst <= 1.0, "tan stays within 1 ulp");
    check(ulps(math.tan(math.inf(1)), math.nan()) == 0.0 && ulps(math.tan(math.nan()), math.nan()) == 0.0 && ulps(math.tan(0.0), 0.0) == 0.0, "tan special values");
}
`, expected: []string{"sin stays within 1 ulp", "sin special values", "cos stays within 1 ulp", "cos special values", "tan stays within 1 ulp", "tan special values"}},

	{dir: "libm_inverse", src: `import "std/math";

float[30] atanIn = [-731.2715117751975, 694.8674738744653, 527.5492379532282, -489.8619485211566, -9.129825816118114, -101.01787042252374, 303.18594544552593, 577.4467022710264, -812.2808264515302, -943.3050469559873, 2.736205631335496, 2.686964922356096, -2.660691793639148, -2.49076802904647, 2.0129932687766976, 1.4158199344111395, 1.0183824086413251, -1.1511812544651348, 2.844107606521153e+78, 8.28228034802315e+39, 7.192673673204811e+124, 8.749002340244788e+54, 0.0, 1.0, -1.0, 0.5, 0.4375, 0.6875, 1.1875, 2.4375];
float[30] atanWant = [-1.5694288465080763, 1.5693572044482045, 1.5689007714033858, -1.5687549381670816, -1.461700103069288, -1.5608974115558094, 1.56749803283117, 1.5690645669694394, -1.569565226093966, -1.5697362247328739, 1.22040713438406, 1.2145115864407445, -1.2112875786393573, -1.1890125084158487, 1.1097339296193796, 0.9558516702506477, 0.794505407067393, -0.8555610530981614, 1.5707963267948966, 1.5707963267948966, 1.5707963267948966, 1.5707963267948966, 0.0, 0.7853981633974483, -0.7853981633974483, 0.4636476090008061, 0.4124104415973873, 0.6022873461349642, 0.870903457075653, 1.1814796049617557];
float[24] asinIn = [0.24580338977940386, 0.4835739785214588, 0.5903871311313933, 0.8849005675541006, 0.4797971494798614, 0.844649993330834, -0.9419895434327705, -0.0687546912437893, 0.8867134339966274, 0.2979491062738484, 0.8018009835012454, -0.7735880706937113, -0.06186190443567252, -0.5068543347603394, 0.08752171847186085, 0.14788237585620156, 0.0, 0.5, -0.5, 0.97, -0.97, 1e-10, 1.0, -1.0];
float[24] asinWant = [0.24834842371258845, 0.5047332593665551, 0.6315384016323191, 1.0862804925997964, 0.5004234973384735, 1.005911001145255, -1.2285094093101632, -0.06880897641716692, 1.09018710876951, 0.3025434578768013, 0.9303028924374376, -0.8844839431292576, -0.061901429043417174, -0.5315317295805175, 0.08763384220406949, 0.14842676190847662, 0.0, 0.5235987755982989, -0.5235987755982989, 1.3252308092796046, -1.3252308092796046, 1e-10, 1.5707963267948966, -1.5707963267948966];
float[24] acosIn = [0.24580338977940386, 0.4835739785214588, 0.5903871311313933, 0.8849005675541006, 0.4797971494798614, 0.844649993330834, -0.9419895434327705, -0.0687546912437893, 0.8867134339966274, 0.2979491062738484, 0.8018009835012454, -0.7735880706937113, -0.06186190443567252, -0.5068543347603394, 0.08752171847186085, 0.14788237585620156, 0.0, 0.5, -0.5, 0.97, -0.97, 1e-10, 1.0, -1.0];
float[24] acosWant = [1.3224479030823082, 1.0660630674283416, 0.9392579251625776, 0.48451583419510014, 1.0703728294564232, 0.5648853256496417, 2.79930573610506, 1.6396053032120634, 0.48060921802538653, 1.2682528689180954, 0.640493434357459, 2.4552802699241543, 1.6326977558383138, 2.102328056375414, 1.4831624845908271, 1.42236956488642, 1.5707963267948966, 1.0471975511965979, 2.0943951023931957, 0.24556551751529213, 2.896027136074501, 1.5707963266948965, 0.0, 3.141592653589793];
float[34] atan2In = [-3.523344703336752, 3.0186894607970753, 0.7176400861337839, -8.840021504505863, -9.250086831160303, -8.602891528507621, -1.5096162171497198, -7.523960777007089, 2.5486644481117864, 1.5420589723499738, 9.525102111858402, 7.1693691809735896, -7.114898332851249, -3.830363517961313, -6.38547240152125, 2.7782693785236816, 0.9548893141911563, -8.807976600675346, 3.6079994636357178, -3.7170565924641696, 6.323241873990504e-13, -7.133631952949054e+51, 3678083.133760015, -2.4670640946315166e+28, 4.976617132840867e-88, -3.948357378673241e-21, 2.185820385317884e-42, -4.2626249349978694e-113, 1.0, -1.0, 0.0, 0.0, 1.0, -1.0];
float[34] atan2In2 = [-6.9830165215099615, -8.551274266649145, -2.6862216617482897, 0.1487146637884056, -1.3270863267522834, -8.185739733122698, 6.537042493440762, -5.535220707859709, 8.954178849140114, -2.066390506984397, -9.068346387644874, -4.207814273366475, -7.644155238432633, 6.322527182400627, 1.6320032732493246, -2.552049145485375, -8.744220500533537, -5.880825743613469, -1.4481538866119426, 1.7112372701527754, -2.7914664083193375e-66, 1.0994187310424542e-109, 6.138946679557191e+59, 2.998092497825765e-100, 2.0667229952156493e-91, 8.72972094338623e+68, 3.895741490952576e+24, 7.473559341346019e+115, 1.0, 1.0, 1.0, -1.0, 0.0, 0.0];
float[34] atan2Want = [-2.674304400263401, 2.8022384108050766, 2.880533522521628, -1.5539750286933314, -1.7132914266533001, -2.3313523233351185, -0.22695395117427608, -2.205068883706394, 0.2773006924279227, 2.500491187878527, 2.3316340177430117, 2.1015393520729835, -2.3920389443185233, -0.5446937228442252, -1.320572210439032, 2.3137796593213746, 3.03282132161895, -2.159493538734648, 1.952485839810284, -1.1393487122193562, 1.5707963267948966, -1.5707963267948966, 5.991391236558711e-54, -1.5707963267948966, 1.570381040100896, -4.5228907135509026e-90, 5.610794223369819e-67, -5.703607531977063e-229, 0.7853981633974483, -0.7853981633974483, 0.0, 3.141592653589793, 1.5707963267948966, -1.5707963267948966];

func ulps(float a, float b) float {
    if math.isNaN(b) {
        if math.isNaN(a) {
            return 0.0;
        }
        return 1000000000.0;
    }
    if math.isNaN(a) {
        return 1000000000.0;
    }
    if math.isInf(b, 0) || math.isInf(a, 0) {
        if a == b {
            return 0.0;
        }
        return 1000000000.0;
    }
    if b == 0.0 {
        if a == 0.0 && math.signbit(a) == math.signbit(b) {
            return 0.0;
        }
        return 1000000000.0;
    }
    let _, e = math.frexp(b);
    float ulp = math.ldexp(1.0, e - 53);
    if ulp < 5e-324 {
        ulp = 5e-324;
    }
    return math.fabs(a - b) / ulp;
}

func main() {
    float atanWorst = 0.0;
    for i in 0..29 {
        float u = ulps(math.atan(atanIn[i]), atanWant[i]);
        if u > atanWorst {
            atanWorst = u;
        }
    }
    check(atanWorst <= 1.0, "atan stays within 1 ulp");
    check(ulps(math.atan(math.inf(1)), 1.5707963267948966) == 0.0 && ulps(math.atan(math.inf(-1)), -1.5707963267948966) == 0.0 && ulps(math.atan(math.nan()), math.nan()) == 0.0 && ulps(math.atan(0.0), 0.0) == 0.0, "atan special values");
    float asinWorst = 0.0;
    for i in 0..23 {
        float u = ulps(math.asin(asinIn[i]), asinWant[i]);
        if u > asinWorst {
            asinWorst = u;
        }
    }
    check(asinWorst <= 1.0, "asin stays within 1 ulp");
    check(ulps(math.asin(1.0), 1.5707963267948966) == 0.0 && ulps(math.asin(-1.0), -1.5707963267948966) == 0.0 && ulps(math.asin(2.0), math.nan()) == 0.0 && ulps(math.asin(math.nan()), math.nan()) == 0.0 && ulps(math.asin(0.0), 0.0) == 0.0, "asin special values");
    float acosWorst = 0.0;
    for i in 0..23 {
        float u = ulps(math.acos(acosIn[i]), acosWant[i]);
        if u > acosWorst {
            acosWorst = u;
        }
    }
    check(acosWorst <= 1.0, "acos stays within 1 ulp");
    check(ulps(math.acos(1.0), 0.0) == 0.0 && ulps(math.acos(-1.0), 3.141592653589793) == 0.0 && ulps(math.acos(2.0), math.nan()) == 0.0 && ulps(math.acos(math.nan()), math.nan()) == 0.0 && ulps(math.acos(0.0), 1.5707963267948966) == 0.0, "acos special values");
    float atan2Worst = 0.0;
    for i in 0..33 {
        float u = ulps(math.atan2(atan2In[i], atan2In2[i]), atan2Want[i]);
        if u > atan2Worst {
            atan2Worst = u;
        }
    }
    check(atan2Worst <= 1.0, "atan2 stays within 1 ulp");
    check(ulps(math.atan2(0.0, 0.0), 0.0) == 0.0 && ulps(math.atan2(0.0, (0.0 - 0.0) * -1.0), 3.141592653589793) == 0.0 && ulps(math.atan2((0.0 - 0.0) * -1.0, 0.0), (0.0 - 0.0) * -1.0) == 0.0 && ulps(math.atan2((0.0 - 0.0) * -1.0, (0.0 - 0.0) * -1.0), -3.141592653589793) == 0.0 && ulps(math.atan2(1.0, math.inf(1)), 0.0) == 0.0 && ulps(math.atan2(1.0, math.inf(-1)), 3.141592653589793) == 0.0 && ulps(math.atan2(math.inf(1), math.inf(1)), 0.7853981633974483) == 0.0 && ulps(math.atan2(math.inf(1), math.inf(-1)), 2.356194490192345) == 0.0 && ulps(math.atan2(math.inf(1), 1.0), 1.5707963267948966) == 0.0 && ulps(math.atan2(math.inf(-1), 1.0), -1.5707963267948966) == 0.0 && ulps(math.atan2(math.nan(), 1.0), math.nan()) == 0.0 && ulps(math.atan2(1.0, math.nan()), math.nan()) == 0.0, "atan2 special values");
}
`, expected: []string{"atan stays within 1 ulp", "atan special values", "asin stays within 1 ulp", "asin special values", "acos stays within 1 ulp", "acos special values", "atan2 stays within 1 ulp", "atan2 special values"}},

	{dir: "libm_hyper", src: `import "std/math";

float[29] sinhIn = [-16.39764843713707, 27.737702150062965, -22.420146080948427, 12.289015372296475, -24.88888391695484, -15.153540904254896, 29.947712349754745, -17.43614208666523, 8.51210610432642, -2.4519742336561983, -2.812054131711431, -0.3010383666493368, -18.46614946543907, 19.8312764179783, -7.39852843699571e-05, -0.0002533761372099159, -0.0007229211749710896, 0.0007331236999726827, -0.0009871298918377534, 5.5641601044167065e-06, 0.0, 1.0, -1.0, 0.5, 22.0, 1e-10, 40.0, 700.0, -700.0];
float[29] sinhWant = [-6612691.113457197, 556289661525.0116, -2728448311.234231, 108647.95805443199, -32216247287.721878, -1905764.357677977, 5071030509526.817, -18680577.507483955, 2487.3144750514853, -5.762562015355917, -8.291995397305712, -0.3056059024602484, -52325913.52831877, 204919799.25699922, -7.398528443745414e-05, -0.00025337613992101806, -0.0007229212379393358, 0.0007331237656447275, -0.0009871300521518383, 5.5641601044454175e-06, 0.0, 1.1752011936438014, -1.1752011936438014, 0.5210953054937474, 1792456423.065796, 1e-10, 1.1769263341851e+17, 5.0711602736750225e+303, -5.0711602736750225e+303];
float[29] coshIn = [-16.39764843713707, 27.737702150062965, -22.420146080948427, 12.289015372296475, -24.88888391695484, -15.153540904254896, 29.947712349754745, -17.43614208666523, 8.51210610432642, -2.4519742336561983, -2.812054131711431, -0.3010383666493368, -18.46614946543907, 19.8312764179783, -7.39852843699571e-05, -0.0002533761372099159, -0.0007229211749710896, 0.0007331236999726827, -0.0009871298918377534, 5.5641601044167065e-06, 0.0, 1.0, -1.0, 0.5, 22.0, 1e-10, 40.0, 700.0, -700.0];
float[29] coshWant = [6612691.113457274, 556289661525.0116, 2728448311.234231, 108647.958059034, 32216247287.721878, 1905764.3576782392, 5071030509526.817, 18680577.507483985, 2487.314676071495, 5.8486854062107705, 8.352076847643293, 1.0456552814472573, 52325913.528318785, 204919799.25699922, 1.0000000027369111, 1.0000000320997335, 1.000000261307524, 1.0000002687351919, 1.0000004872127513, 1.0000000000154798, 1.0, 1.5430806348152437, 1.5430806348152437, 1.1276259652063807, 1792456423.065796, 1.0, 1.1769263341851e+17, 5.0711602736750225e+303, 5.0711602736750225e+303];
float[29] tanhIn = [-16.39764843713707, 27.737702150062965, -22.420146080948427, 12.289015372296475, -24.88888391695484, -15.153540904254896, 29.947712349754745, -17.43614208666523, 8.51210610432642, -2.4519742336561983, -2.812054131711431, -0.3010383666493368, -18.46614946543907, 19.8312764179783, -7.39852843699571e-05, -0.0002533761372099159, -0.0007229211749710896, 0.0007331236999726827, -0.0009871298918377534, 5.5641601044167065e-06, 0.0, 1.0, -1.0, 0.5, 22.0, 1e-10, 40.0, 700.0, -700.0];
float[29] tanhWant = [-0.9999999999999886, 1.0, -1.0, 0.9999999999576429, -1.0, -0.9999999999998623, 1.0, -0.9999999999999986, 0.9999999191819147, -0.985274743831597, -0.9928064059474581, -0.29226257245817117, -0.9999999999999998, 1.0, -7.398528423496299e-05, -0.0002533761317877117, -0.0007229210490346266, 0.0007331235686286247, -0.000987129571209724, 5.564160104359284e-06, 0.0, 0.7615941559557649, -0.7615941559557649, 0.46211715726000974, 1.0, 1e-10, 1.0, 1.0, -1.0];
float[29] cbrtIn = [142805.1893798269, -142221.89064977074, 156182.60226894077, -587803.5357209966, 626642.50271464, 647177.745066891, 306945.06780235143, -679540.8869623607, 41338.71927984909, -344454.3767558137, 2.6775161921773473e-29, 7.30182883550128e+35, 3.360086163397612e+254, 2.454946656957562e-21, 50671.50894967005, 2.6971016203361873e+52, 6.254697535355299e-190, 13969582.443943003, 8.504173854606704e+77, 6.1111579958679464e+175, 0.0, 1.0, -1.0, 8.0, -27.0, 1e-310, 5e-324, 1e+308, 64.0];
float[29] cbrtWant = [52.2694579672263, -52.19819465110767, 53.85312192319572, -83.76785561677048, 85.57362888976633, 86.49835697222827, 67.45594328937398, -87.91679838136442, 34.57686963530208, -70.09879768605914, 2.9916494413930284e-10, 900486520304.3416, 6.952112715111348e+84, 1.3490064307013428e-07, 37.004506134817554, 2.998926141705759e+17, 8.552021242940491e-64, 240.83955071469737, 9.474232612219239e+25, 3.938895907977069e+58, 0.0, 1.0, -1.0, 2.0, -3.0, 4.641588833612774e-104, 1.7031839360032603e-108, 4.641588833612779e+102, 4.0];
float[27] hypotIn = [-5.085864262290386, 33.28209422496761, -97.82791138198641, -45.19037210433372, 38.118530607736915, 11.638009550669608, -70.9394410579794, -67.54653552222227, -88.2351345178268, -85.07807327282451, -32.60008144164648, 68.48067356251235, -87.84306076220622, 1.785183285540299, 1.5256171878968414e+296, 3.7085107648648757e+286, -3.2178485129626407e+289, 3.266984050326304e-61, 7.495622743879694e-15, 1.6624120732911055e+284, 4.0986770555958134e+222, 2.7707441537506968e-151, 3.0, 0.0, 1e+308, 1e-310, 5e-324];
float[27] hypotIn2 = [31.494500531451052, -71.47992941492645, -25.049101587327115, 62.06961044701674, 20.291407745546636, 32.26421717440019, -11.989041615046034, 81.19459630586451, 63.76402556841606, 37.38910081870836, -19.07714337427018, -96.27913993759944, 83.00672868198939, -81.80440218828333, 2.582537783336413e-236, -1.89821274679496e+232, -3.404767557015227e+33, 1.7243067365956842e+102, -1.0451907991655672e+25, 4.5301534182641625e-88, -4.5290651270709035e+105, -7.912784724479016e-84, 4.0, 0.0, 1e+308, 1e-310, 5e-324];
float[27] hypotWant = [31.902501140506416, 78.84845023944662, 100.98394791101218, 76.77764174531902, 43.182908702274865, 34.299022962445115, 71.94540580513699, 105.6176923168043, 108.86362992326045, 92.93128327881648, 37.77171837929066, 118.1493776483788, 120.85743804399485, 81.82387852421435, 1.5256171878968414e+296, 3.7085107648648757e+286, 3.2178485129626407e+289, 1.7243067365956842e+102, 1.0451907991655672e+25, 1.6624120732911055e+284, 4.0986770555958134e+222, 7.912784724479016e-84, 5.0, 0.0, 1.4142135623730951e+308, 1.4142135623731e-310, 5e-324];

func ulps(float a, float b) float {
    if math.isNaN(b) {
        if math.isNaN(a) {
            return 0.0;
        }
        return 1000000000.0;
    }
    if math.isNaN(a) {
        return 1000000000.0;
    }
    if math.isInf(b, 0) || math.isInf(a, 0) {
        if a == b {
            return 0.0;
        }
        return 1000000000.0;
    }
    if b == 0.0 {
        if a == 0.0 && math.signbit(a) == math.signbit(b) {
            return 0.0;
        }
        return 1000000000.0;
    }
    let _, e = math.frexp(b);
    float ulp = math.ldexp(1.0, e - 53);
    if ulp < 5e-324 {
        ulp = 5e-324;
    }
    return math.fabs(a - b) / ulp;
}

func main() {
    float sinhWorst = 0.0;
    for i in 0..28 {
        float u = ulps(math.sinh(sinhIn[i]), sinhWant[i]);
        if u > sinhWorst {
            sinhWorst = u;
        }
    }
    check(sinhWorst <= 2.0, "sinh stays within 2 ulp");
    check(ulps(math.sinh(math.inf(1)), math.inf(1)) == 0.0 && ulps(math.sinh(math.inf(-1)), math.inf(-1)) == 0.0 && ulps(math.sinh(math.nan()), math.nan()) == 0.0 && ulps(math.sinh(0.0), 0.0) == 0.0 && ulps(math.sinh(711.0), math.inf(1)) == 0.0, "sinh special values");
    float coshWorst = 0.0;
    for i in 0..28 {
        float u = ulps(math.cosh(coshIn[i]), coshWant[i]);
        if u > coshWorst {
            coshWorst = u;
        }
    }
    check(coshWorst <= 2.0, "cosh stays within 2 ulp");
    check(ulps(math.cosh(math.inf(1)), math.inf(1)) == 0.0 && ulps(math.cosh(math.inf(-1)), math.inf(1)) == 0.0 && ulps(math.cosh(math.nan()), math.nan()) == 0.0 && ulps(math.cosh(0.0), 1.0) == 0.0 && ulps(math.cosh(711.0), math.inf(1)) == 0.0, "cosh special values");
    float tanhWorst = 0.0;
    for i in 0..28 {
        float u = ulps(math.tanh(tanhIn[i]), tanhWant[i]);
        if u > tanhWorst {
            tanhWorst = u;
        }
    }
    check(tanhWorst <= 2.0, "tanh stays within 2 ulp");
    check(ulps(math.tanh(math.inf(1)), 1.0) == 0.0 && ulps(math.tanh(math.inf(-1)), -1.0) == 0.0 && ulps(math.tanh(math.nan()), math.nan()) == 0.0 && ulps(math.tanh(0.0), 0.0) == 0.0 && ulps(math.tanh(40.0), 1.0) == 0.0, "tanh special values");
    float cbrtWorst = 0.0;
    for i in 0..28 {
        float u = ulps(math.cbrt(cbrtIn[i]), cbrtWant[i]);
        if u > cbrtWorst {
            cbrtWorst = u;
        }
    }
    check(cbrtWorst <= 1.0, "cbrt stays within 1 ulp");
    check(ulps(math.cbrt(math.inf(1)), math.inf(1)) == 0.0 && ulps(math.cbrt(math.inf(-1)), math.inf(-1)) == 0.0 && ulps(math.cbrt(math.nan()), math.nan()) == 0.0 && ulps(math.cbrt(0.0), 0.0) == 0.0 && ulps(math.cbrt(27.0), 3.0) == 0.0 && ulps(math.cbrt(-8.0), -2.0) == 0.0, "cbrt special values");
    float hypotWorst = 0.0;
    for i in 0..26 {
        float u = ulps(math.hypot(hypotIn[i], hypotIn2[i]), hypotWant[i]);
        if u > hypotWorst {
            hypotWorst = u;
        }
    }
    check(hypotWorst <= 1.0, "hypot stays within 1 ulp");
    check(ulps(math.hypot(math.inf(1), math.nan()), math.inf(1)) == 0.0 && ulps(math.hypot(math.nan(), math.inf(1)), math.inf(1)) == 0.0 && ulps(math.hypot(math.nan(), 1.0), math.nan()) == 0.0 && ulps(math.hypot(3.0, 4.0), 5.0) == 0.0 && ulps(math.hypot(0.0, 0.0), 0.0) == 0.0, "hypot special values");
}
`, expected: []string{"sinh stays within 2 ulp", "sinh special values", "cosh stays within 2 ulp", "cosh special values", "tanh stays within 2 ulp", "tanh special values", "cbrt stays within 1 ulp", "cbrt special values", "hypot stays within 1 ulp", "hypot special values"}},

	{dir: "libm_fmod", src: `import "std/math";

float[13] fmodIn = [5.5, -5.5, 5.5, 1e+300, 1e+300, 123456789.123, 2.5e-323, 1.7976931348623157e+308, 0.1, 3.0, 1e+22, 7.0, 2.5e-323];
float[13] fmodIn2 = [2.0, 2.0, -2.0, 7.0, 1e-300, 0.001, 1e-323, 3.0, 0.03, 5.0, 10.0, 7.0, 1e-323];
float[13] fmodWant = [1.5, -1.5, 1.5, 1.0, 4.891554850853602e-301, 0.000999993376923471, 5e-324, 2.0, 0.010000000000000009, 3.0, 0.0, 0.0, 5e-324];

func ulps(float a, float b) float {
    if math.isNaN(b) {
        if math.isNaN(a) {
            return 0.0;
        }
        return 1000000000.0;
    }
    if math.isNaN(a) {
        return 1000000000.0;
    }
    if math.isInf(b, 0) || math.isInf(a, 0) {
        if a == b {
            return 0.0;
        }
        return 1000000000.0;
    }
    if b == 0.0 {
        if a == 0.0 && math.signbit(a) == math.signbit(b) {
            return 0.0;
        }
        return 1000000000.0;
    }
    let _, e = math.frexp(b);
    float ulp = math.ldexp(1.0, e - 53);
    if ulp < 5e-324 {
        ulp = 5e-324;
    }
    return math.fabs(a - b) / ulp;
}

func main() {
    float fmodWorst = 0.0;
    for i in 0..12 {
        float u = ulps(math.fmod(fmodIn[i], fmodIn2[i]), fmodWant[i]);
        if u > fmodWorst {
            fmodWorst = u;
        }
    }
    check(fmodWorst <= 0.0, "fmod stays within 0 ulp");
    check(ulps(math.fmod(7.0, 7.0), 0.0) == 0.0 && ulps(math.fmod(3.0, 5.0), 3.0) == 0.0 && ulps(math.fmod(1.0, math.inf(1)), 1.0) == 0.0 && ulps(math.fmod(-1.0, math.inf(1)), -1.0) == 0.0, "fmod special values");
}
`, expected: []string{"fmod stays within 0 ulp", "fmod special values"}},

	{dir: "conv_float", src: `import "std/conv";
import "std/math";

float[228] xs = [1.0, 0.5, 0.1, 0.2, 0.3, 1e+23, 1e+22, 1e+21, 1e+20, 1e-05, 1e-06, 1e-07, 0.000123, 5e-324, 2.2250738585072014e-308, 1.7976931348623157e+308, 123456.789, 100.0, 3.14159, 2.718281828459045, 9.995, 0.375, 0.125, 2.5, 1.5, 0.999, 99.5, 1000000000000000.0, 1.2345678901234568e+17, 1e+23, 1.152921504606847e+18, 4503599627370496.0, 0.05, 0.25, 4.35, 0.0001, 123.456, 9007199254740992.0, 0.0, -0.0, -1.0, -0.5, -0.1, -0.2, -0.3, -1e+23, -1e+22, -1e+21, -1e+20, -1e-05, -1e-06, -1e-07, -0.000123, -5e-324, -2.2250738585072014e-308, -1.7976931348623157e+308, 5e-324, 4.144523e-317, 3.4766779039175e-310, 2.916448807822559e-303, 2.446494580089078e-296, 2.0522684006491881e-289, 1.7215675123832985e-282, 1.4441555006918637e-275, 1.2114454386347773e-268, 1.0162340898095202e-261, 8.52478941564886e-255, 7.151111669042735e-248, 5.998787255582524e-241, 5.0321474762477604e-234, 4.2212712576431773e-227, 3.541058984203562e-220, 2.9704555723361872e-213, 2.491798737774392e-206, 2.0902722826084166e-199, 1.7534474792067224e-192, 1.4708983551653345e-185, 1.2338789709326767e-178, 1.035052700659762e-171, 8.682651365176084e-165, 7.283535870312702e-158, 6.10987272699921e-151, 5.1253327236687384e-144, 4.299440708842937e-137, 3.606632272572553e-130, 3.02546243347603e-123, 2.5379418373156492e-116, 2.1289799200040754e-109, 1.7859177988785547e-102, 1.4981364335015035e-95, 1.256727927116218e-88, 1.0542197943230523e-81, 8.843436600416711e-75, 7.418412301374843e-68, 6.223015277861142e-61, 5.22024357439882e-54, 4.3790577010150533e-47, 3.6734198463196485e-40, 3.0814879110195774e-33, 2.5849394142282115e-26, 2.168404344971009e-19, 1.8189894035458565e-12, 1.52587890625e-05, 128.0, 1073741824.0, 9007199254740992.0, 7.555786372591432e+22, 6.338253001141147e+29, 5.316911983139664e+36, 4.460149039706125e+43, 3.7414441915671115e+50, 3.1385508676933404e+57, 2.6328072917139297e+64, 2.2085588309729804e+71, 1.8526734277970591e+78, 1.5541351137805833e+85, 1.3037030248540711e+92, 1.093625362391506e+99, 9.173994463960286e+105, 7.695704335233297e+112, 6.455624695217272e+119, 5.415370496329717e+126, 4.542742026847543e+133, 3.8107282108349515e+140, 3.196670515523576e+147, 2.6815615859885194e+154, 2.2494568972715982e+161, 1.8869812124107707e+168, 1.582914569427869e+175, 1.3278449820419177e+182, 1.1138771039116688e+189, 9.343878384890256e+195, 7.838213297051748e+202, 6.575169876935467e+209, 5.515652263101987e+216, 4.626864469947544e+223, 3.8812952307517723e+230, 3.2558664223046163e+237, 2.7312187117075883e+244, 2.291112313477997e+251, 1.9219243081740033e+258, 1.612226962694291e+265, 1.352433999707303e+272, 1.134503866941668e+279, 9.516908214257812e+285, 7.98336123813888e+292, 6.696928794914171e+299, 5.617791046444737e+306, 1.0846593320521653e-111, 3.827838586498353e+278, 1.1911478289890319e+91, 3.7276561872477e+279, 1.3653007914819919e+237, 5.3429589226471036e+243, 1.6278205303253624e+75, 2.4280161293618904e-49, 1.1567257858343133e+45, 2.606387362110178e+211, 6.862662879744832e+170, 5.234574084589718e-06, 4.7289621559886885e+168, 1.0250718162476351e-36, 4.2413611217723204e-307, 6.878915419638227e-259, 1.4290740184757722e-131, 3.235378915349377e-248, 1.4270773302164413e-301, 2.258599388854955e+241, 2.1118825046470448e-06, 6.291845313562883e-115, 9.123292170577517e-64, 2.4048551901096585e-94, 7.936797034399582e+307, 8.738245870117971e-89, 1.2114921281708982e-76, 9.146578401575517e+151, 1.624623429291922e+85, 7.194724112637607e+137, 2.2285169258594296e-253, 5.913330239262773e+23, 2.273918926158149e-131, 8.517071473104096e-222, 3.921932509748071e+89, 5.535950019483495e+133, 3.6271939668999635e-118, 5.527104338142335e+214, 2.9386314651465816e-81, 2.1118269707776657e-24, 43.70225546358949, 8.444807665433402e+17, 1.2637405834466526e+17, 36632.82207284301, 6.3717448457681405, 2.0479826081217932e+16, -62065.991241516494, -9260783.72451528, -2488797300621367.0, 781594735696.8141, 2004.6520687610835, -4.955734401956704e+17, 554535058.259937, 25052839446.311405, 2633131.1699909396, 1037041.6973867594, 863093714258093.6, 104.09943247115632, -7452753710206333.0, 1.3228152254733891e-12, 774900025230436.1, -0.29965278369793547, 0.00010193669270915556, 1529.1825692571877, -1102924998815943.6, 7195511241376391.0, -744.3015475397682, 160.1455926829016, -509.61541850508615, 3.779988296964296e+20, 4.3141735112427586e+19, 2.2581887956871841e+18, 2.1358283193825795e-10, -1.1381169656542993e-08, 1.4908423020533698e-10, -13952920.239815185, -147700579.50750175, -3.818792909503528e-05, 12396067411339.193, 15719542003.269464];
int[228] ns = [2, 0, 5, 10, 9, 9, 12, 7, 11, 11, 2, 8, 3, 7, 9, 2, 9, 11, 8, 4, 1, 6, 10, 2, 2, 12, 12, 9, 4, 8, 6, 6, 12, 2, 8, 12, 3, 6, 0, 1, 6, 1, 0, 5, 10, 5, 12, 0, 0, 8, 7, 5, 10, 4, 7, 5, 1, 7, 7, 9, 9, 12, 2, 2, 9, 8, 3, 6, 9, 5, 7, 9, 3, 10, 11, 1, 8, 6, 0, 3, 10, 2, 0, 10, 8, 0, 7, 12, 10, 3, 7, 11, 10, 0, 2, 10, 8, 3, 12, 8, 2, 2, 10, 2, 6, 4, 6, 8, 6, 1, 6, 4, 2, 5, 2, 4, 7, 7, 12, 6, 4, 7, 2, 5, 8, 7, 7, 1, 8, 12, 8, 12, 4, 1, 3, 3, 0, 1, 7, 10, 5, 6, 8, 12, 10, 2, 0, 5, 2, 5, 8, 10, 9, 12, 8, 3, 1, 9, 11, 4, 8, 12, 9, 11, 3, 9, 4, 9, 4, 10, 8, 2, 1, 12, 4, 1, 6, 2, 1, 10, 0, 12, 5, 11, 12, 2, 10, 10, 1, 8, 5, 4, 9, 1, 4, 5, 2, 4, 4, 1, 0, 6, 12, 4, 6, 0, 0, 1, 3, 0, 4, 1, 1, 10, 8, 9, 5, 5, 12, 7, 9, 10, 10, 11, 9, 0, 6, 5];
string[228] shortest = ["1", "0.5", "0.1", "0.2", "0.3", "1e+23", "1e+22", "1e+21", "100000000000000000000", "0.00001", "1e-06", "1e-07", "0.000123", "5e-324", "2.2250738585072014e-308", "1.7976931348623157e+308", "123456.789", "100", "3.14159", "2.718281828459045", "9.995", "0.375", "0.125", "2.5", "1.5", "0.999", "99.5", "1000000000000000", "123456789012345680", "1e+23", "1152921504606847000", "4503599627370496", "0.05", "0.25", "4.35", "0.0001", "123.456", "9007199254740992", "0", "-0", "-1", "-0.5", "-0.1", "-0.2", "-0.3", "-1e+23", "-1e+22", "-1e+21", "-100000000000000000000", "-0.00001", "-1e-06", "-1e-07", "-0.000123", "-5e-324", "-2.2250738585072014e-308", "-1.7976931348623157e+308", "5e-324", "4.144523e-317", "3.4766779039175e-310", "2.916448807822559e-303", "2.446494580089078e-296", "2.0522684006491881e-289", "1.7215675123832985e-282", "1.4441555006918637e-275", "1.2114454386347773e-268", "1.0162340898095202e-261", "8.52478941564886e-255", "7.151111669042735e-248", "5.998787255582524e-241", "5.0321474762477604e-234", "4.2212712576431773e-227", "3.541058984203562e-220", "2.9704555723361872e-213", "2.491798737774392e-206", "2.0902722826084166e-199", "1.7534474792067224e-192", "1.4708983551653345e-185", "1.2338789709326767e-178", "1.035052700659762e-171", "8.682651365176084e-165", "7.283535870312702e-158", "6.10987272699921e-151", "5.1253327236687384e-144", "4.299440708842937e-137", "3.606632272572553e-130", "3.02546243347603e-123", "2.5379418373156492e-116", "2.1289799200040754e-109", "1.7859177988785547e-102", "1.4981364335015035e-95", "1.256727927116218e-88", "1.0542197943230523e-81", "8.843436600416711e-75", "7.418412301374843e-68", "6.223015277861142e-61", "5.22024357439882e-54", "4.3790577010150533e-47", "3.6734198463196485e-40", "3.0814879110195774e-33", "2.5849394142282115e-26", "2.168404344971009e-19", "1.8189894035458565e-12", "0.0000152587890625", "128", "1073741824", "9007199254740992", "7.555786372591432e+22", "6.338253001141147e+29", "5.316911983139664e+36", "4.460149039706125e+43", "3.7414441915671115e+50", "3.1385508676933404e+57", "2.6328072917139297e+64", "2.2085588309729804e+71", "1.8526734277970591e+78", "1.5541351137805833e+85", "1.3037030248540711e+92", "1.093625362391506e+99", "9.173994463960286e+105", "7.695704335233297e+112", "6.455624695217272e+119", "5.415370496329717e+126", "4.542742026847543e+133", "3.8107282108349515e+140", "3.196670515523576e+147", "2.6815615859885194e+154", "2.2494568972715982e+161", "1.8869812124107707e+168", "1.582914569427869e+175", "1.3278449820419177e+182", "1.1138771039116688e+189", "9.343878384890256e+195", "7.838213297051748e+202", "6.575169876935467e+209", "5.515652263101987e+216", "4.626864469947544e+223", "3.8812952307517723e+230", "3.2558664223046163e+237", "2.7312187117075883e+244", "2.291112313477997e+251", "1.9219243081740033e+258", "1.612226962694291e+265", "1.352433999707303e+272", "1.134503866941668e+279", "9.516908214257812e+285", "7.98336123813888e+292", "6.696928794914171e+299", "5.617791046444737e+306", "1.0846593320521653e-111", "3.827838586498353e+278", "1.1911478289890319e+91", "3.7276561872477e+279", "1.3653007914819919e+237", "5.3429589226471036e+243", "1.6278205303253624e+75", "2.4280161293618904e-49", "1.1567257858343133e+45", "2.606387362110178e+211", "6.862662879744832e+170", "5.234574084589718e-06", "4.7289621559886885e+168", "1.0250718162476351e-36", "4.2413611217723204e-307", "6.878915419638227e-259", "1.4290740184757722e-131", "3.235378915349377e-248", "1.4270773302164413e-301", "2.258599388854955e+241", "2.1118825046470448e-06", "6.291845313562883e-115", "9.123292170577517e-64", "2.4048551901096585e-94", "7.936797034399582e+307", "8.738245870117971e-89", "1.2114921281708982e-76", "9.146578401575517e+151", "1.624623429291922e+85", "7.194724112637607e+137", "2.2285169258594296e-253", "5.913330239262773e+23", "2.273918926158149e-131", "8.517071473104096e-222", "3.921932509748071e+89", "5.535950019483495e+133", "3.6271939668999635e-118", "5.527104338142335e+214", "2.9386314651465816e-81", "2.1118269707776657e-24", "43.70225546358949", "844480766543340200", "126374058344665260", "36632.82207284301", "6.3717448457681405", "20479826081217932", "-62065.991241516494", "-9260783.72451528", "-2488797300621367", "781594735696.8141", "2004.6520687610835", "-495573440195670400", "554535058.259937", "25052839446.311405", "2633131.1699909396", "1037041.6973867594", "863093714258093.6", "104.09943247115632", "-7452753710206333", "1.3228152254733891e-12", "774900025230436.1", "-0.29965278369793547", "0.00010193669270915556", "1529.1825692571877", "-1102924998815943.6", "7195511241376391", "-744.3015475397682", "160.1455926829016", "-509.61541850508615", "377998829696429600000", "43141735112427586000", "2258188795687184100", "2.1358283193825795e-10", "-1.1381169656542993e-08", "1.4908423020533698e-10", "-13952920.239815185", "-147700579.50750175", "-0.00003818792909503528", "12396067411339.193", "15719542003.269464"];
string[228] fixed = ["1.00", "0", "0.10000", "0.2000000000", "0.300000000", "99999999999999991611392.000000000", "10000000000000000000000.000000000000", "1000000000000000000000.0000000", "100000000000000000000.00000000000", "0.00001000000", "0.00", "0.00000010", "0.000", "0.0000000", "0.000000000", "179769313486231570814527423731704356798070567525844996598917476803157260780028538760589558632766878171540458953514382464234321326889464182768467546703537516986049910576551282076245490090389328944075868508455133942304583236903222948165808559332123348274797826204144723168738177180919299881250404026184124858368.00", "123456.789000000", "100.00000000000", "3.14159000", "2.7183", "10.0", "0.375000", "0.1250000000", "2.50", "1.50", "0.999000000000", "99.500000000000", "1000000000000000.000000000", "123456789012345680.0000", "99999999999999991611392.00000000", "1152921504606846976.000000", "4503599627370496.000000", "0.050000000000", "0.25", "4.35000000", "0.000100000000", "123.456", "9007199254740992.000000", "0", "-0.0", "-1.000000", "-0.5", "-0", "-0.20000", "-0.3000000000", "-99999999999999991611392.00000", "-10000000000000000000000.000000000000", "-1000000000000000000000", "-100000000000000000000", "-0.00001000", "-0.0000010", "-0.00000", "-0.0001230000", "-0.0000", "-0.0000000", "-179769313486231570814527423731704356798070567525844996598917476803157260780028538760589558632766878171540458953514382464234321326889464182768467546703537516986049910576551282076245490090389328944075868508455133942304583236903222948165808559332123348274797826204144723168738177180919299881250404026184124858368.00000", "0.0", "0.0000000", "0.0000000", "0.000000000", "0.000000000", "0.000000000000", "0.00", "0.00", "0.000000000", "0.00000000", "0.000", "0.000000", "0.000000000", "0.00000", "0.0000000", "0.000000000", "0.000", "0.0000000000", "0.00000000000", "0.0", "0.00000000", "0.000000", "0", "0.000", "0.0000000000", "0.00", "0", "0.0000000000", "0.00000000", "0", "0.0000000", "0.000000000000", "0.0000000000", "0.000", "0.0000000", "0.00000000000", "0.0000000000", "0", "0.00", "0.0000000000", "0.00000000", "0.000", "0.000000000000", "0.00000000", "0.00", "0.00", "0.0000152588", "128.00", "1073741824.000000", "9007199254740992.0000", "75557863725914323419136.000000", "633825300114114700748351602688.00000000", "5316911983139663491615228241121378304.000000", "44601490397061246283071436545296723011960832.0", "374144419156711147060143317175368453031918731001856.000000", "3138550867693340381917894711603833208051177722232017256448.0000", "26328072917139296674479506920917608079723773850137277813577744384.00", "220855883097298041197912187592864814478435487109452369765200775161577472.00000", "1852673427797059126777135760139006525652319754650249024631321344126610074238976.00", "15541351137805832567355695254588151253139254712417116170014499277911234281641667985408.0000", "130370302485407109521180524058200202307293977194619920040712988758680403184853549195737432064.0000000", "1093625362391505962186251113558810682676584715446606218212885303204976499599687961611756588511526912.0000000", "9173994463960286046443283581208347763186259956673124494950355357547691504353939232280074212440502746218496.000000000000", "76957043352332967211482500195592995713046365762627825523336510555167425334955489475418488779072100860950445293568.000000", "645562469521727147413979793000752968582426448207305878207664839135161905504210298657411338320034457858975792993186873344.0000", "5415370496329716522614090203404460358274291162843391748379842930887932241807862544999950011922147613471467208908991351228465152.0000000", "45427420268475430659332737993000283397102585042957378767593137448789955507087370207886940669610222847547657600391636120845912601788416.00", "381072821083495145432323880589986121307201921712032611188861933548019011086397170424842053596617672260721060927906081896416989218663120764928.00000", "3196670515523576044934755563308202297086564498088930458479776726656380660551439995003193449537015778467662777468320381844938727095591204153641140224.00000000", "26815615859885194199148049996411692254958731641184786755447122887443528060147093953603748596333806855380063716372972101707507765623893139892867298012168192.0000000", "224945689727159819140526925384299092943484855915095831655037778630591879033574393515952034305194542857496045531676044756160413302774714984450425759043258192756736.0000000", "1886981212410770676120777290494134445458460610208220214188103150122812081196074426043063362588829383770734187515381922449885292314962396316280717125716348021824697663488.0", "15829145694278690179872161345257420162248406342477557754500035589945422406818039498900249667919554879334250971265032917718487362195632077437922953926521162746662833417516744704.00000000", "132784498204191774672397051638117156832398279431757980799861034550100889965213060684790625566307321417222332371561625253836644834413176809852379994691646837985957817708848304757932032.000000000000", "1113877103911668754551067286547922686741510866027480451801560673315252726369306002564920119950530126899082595110740822097336109551117050292542153642510306198303709637294986560078826707091456.00000000", "9343878384890255807777119448474196633381331982845050737826186276657715542443371287564109437577976626746659450006721346172290467269376897020421450382791094657540085093089822617769726345721044533248.000000000000", "78382132970517478991165606422426233672355708521949855379734648809861125861064803929830588940942115354940041435601982738271645192059633193360683542052684438972998018132574030730004068581526319939960438784.0000", "657516987693546688405123735360016083193792475352896732437285112883591519287135102764208317034698556403372871107022277214127429031272975482890976846331485106244402958891055574773957969735540339658911656466972672.0", "5515652263101987298728728207430913795608113109085112352897269396216198887424215820128660001943808587833784893551335930816647064191168732319583111500951066614122648616177179922993422016587311577585463592732098692120576.000", "46268644699475435470014199270680622913148582491776246164412857235254375716637876222457838321585848270371190628323884999935972095850551557285913445801770125007762163162852820919462003875720454598226040577701224945512200798208.000", "388129523075177233787244872115625638814221504279174152784763009506512738171594221582719602207161619487621932674282768301542895011028703597861071818760295284801113744005212476387566321407899611206315749798429117187723211713454014464", "3255866422304616344765552632188114158762089024568314531443485259650408807528140659922574316831813618526821245406949824436469141432675471230646169947427163108444901161872077421124549944292877941762189949285100879873872435565174053364826112.0", "27312187117075883106631072934794271937104930088205959824983071996985496526101020964951786294769886374883040913790702552866360531575272919369168306390347079868805765445689403631464768659094790245289840706092591521717005303961503585448607241732096.0000000", "229111231347799689635350271469398707925773913377365220235531597618488512042823233274762274126597027003434876073751997760595174886056587013603560208332516636964102994463833696818134410091831830209960320065853961979807444428853900668922870296851794362368.0000000000", "1921924308174003258872616370050369756495810455948672905389542244035233680030523317234537010836564833497229828911284598430510768810573374275010933992099815720970170092183271059798176857571621225553914820586983072295508566795439261982531741155133337002515103744.00000", "16122269626942909129404900662735492142298807557254685123533957184651913530173488143140175045399694454793530120643833272670970079330526292030350920973600450955456136596649325078391464772840162385651374295294530896122681527488756156580761624107880751845994219387748352.000000", "135243399970730303066198984938628251268824915465247109664757941510828519054521564025450817507240240101036645118273824941793880975232687497536137978486512531688546991103745301547195268525165168909574203624502064231461895242944399805142629606084361337981321876709820927574016.00000000", "1134503866941667986143541334648056457619674836471095626110665786221268201568912028156408931347775536033476809532312754097331684299884724203331627335435786915422798797948806666521214607112348737236205441118127012028555106130125335760617903854636122194680820545543017511614811209728.000000000000", "9516908214257811601907599988159363584840065290620124537956939899622020205826587990689077212775400643774711832257235027522909345571487396529861315719055325605011013378863743193233193022939505515969530853007049198118833591724018432564205433218231411731277088674906521042072098232413978624.0000000000", "79833612381388792466254908521473542642698050417418301660101889697488475674758562631398318619705428043573697873767699809758897519535743746429566871931393256812800226918043427060701508857774519487306134269781756959733232418204834815555554210737921766300284836275030241665694339809213760397115392.00", "669692879491417075592765655662501131600878007315958504652343992731469406953085076558248986759809911329746670573470716765741965803557696277249036098418660925245910485926514436588817162816398196367372136384565404686473871329212422972447846496629816432160699779855408885478776864478289024177325354254336", "5617791046444737211654078721215702292556178059194708039794690036179146118921905097897139916325235500660003558745981042426837180275450519452901482207483566386805246669527046414884444362538940441232908842252656430276192208823201965046059784704400851161354703458893321819998351435577491134526104885300757004288.00000", "0.00", "382783858649835312998059259379585962980913189259961295132363681229404288120567225429229154342162481344861653071091598921429610309531815629185987858938229715130049285857112935446778149062652733681940819897924061405198195364816739294631193147518072471542487011264436470444795625472.00000", "11911478289890318933233656150104484001390591296674155805698192564751847790054903973261869056.00000000", "3727656187247700127713408548141998181126772038330437249250339547845364428074236473021086879735793504351992627574511132467582440501754539787490721169392814939052663536886970556379920949920518842920483883127880269424163197375922079893559024095859017487165013737675828299334821085184.0000000000", "1365300791481991878992475655405618405582440136977125424926971489685800651525539777855917178375630894558529245847619210147540696285334633137814898919385845308154935348776825233136335926432505595594349365073160840123031238266439622557958144.000000000", "5342958922647103554024987344105433580732544161049034675700983701092003762182878506031917881445115316332907219192288254340802601925502889783917630673913012179441680643249176484819688072537971967147767876209632395623645012061707666811586319745024.000000000000", "1627820530325362416684850076204815261111420869873146256810160292807374274560.00000000", "0.000", "1156725785834313301033035650532098759932772352.0", "26063873621101780268869868981214323994666451212955085871823171916661954314043401687198964624681946080652424907360250915664814293260787236379182248904259936225265352374316202425210092928501683857336405356124831744.000000000", "686266287974483162755661044886134755731078545643452547896130754664215772946589025072285125019648167861187138238840616007995067388027003711068728983757002571868252944728064.00000000000", "0.0000", "4728962155988688516462373293219827002913707667984768860674392372937707198568037868485128069164349671878523361767190321269635016556821855571254520307372307031298345533440.00000000", "0.000000000000", "0.000000000", "0.00000000000", "0.000", "0.000000000", "0.0000", "22585993888549549985589534450889036275863823839510900961757628826356293632822262887007785874963840459868648360526743242109997045703674718446864112930115449557202349788080187522209332934315775726928204293190269564675334868526030779043259875328.000000000", "0.0000", "0.0000000000", "0.00000000", "0.00", "79367970343995815212900396243250898913775880859213761097442913048039514289246594857418546716368415884852840718465949679251265606722558597093433645432422696756219012555276178720782031588780863236928307347582430614324596887781449212508657753708294399499338466830257940140587728047744485667397344683170103033856.0", "0.000000000000", "0.0000", "91465784015755166126930680949586216749485971686400238109146920719159139083897984043240513806389380556138032771898731389781950595818335628257427437125632.0", "16246234292919218822453163570986126592399715739278405378137178018491844796692601765888.000000", "719472411263760722278169518203223825052287375864630765001261451965240713805915384886765680361121785228447541818726264079930119486057742336.00", "0.0", "591333023926277304745984.0000000000", "0", "0.000000000000", "392193250974807083701903137234515081168852554249698148968032824231634747046735764745355264.00000", "55359500194834945684636719407020312085277404183775610843683302844633076514531137110368135714716130933215573933169838780453820156608512.00000000000", "0.000000000000", "55271043381423348927823682873502364624917743410175942309404460112914516499672226128183400855713443031605943576933745105555893355377210164760931147222964439831303386236532046725014364605733815803466864104225017692160.00", "0.0000000000", "0.0000000000", "43.7", "844480766543340160.00000000", "126374058344665264.00000", "36632.8221", "6.371744846", "20479826081217932.0", "-62065.9912", "-9260783.72452", "-2488797300621367.00", "781594735696.8141", "2004.6521", "-495573440195670400.0", "554535058", "25052839446.311405", "2633131.169990939554", "1037041.6974", "863093714258093.625000", "104", "-7452753710206333", "0.0", "774900025230436.125", "-0", "0.0001", "1529.2", "-1102924998815943.6", "7195511241376391.0000000000", "-744.30154754", "160.145592683", "-509.61542", "377998829696429588480.00000", "43141735112427585536.000000000000", "2258188795687184128.0000000", "0.000000000", "-0.0000000114", "0.0000000001", "-13952920.23981518485", "-147700579.507501751", "-0", "12396067411339.193359", "15719542003.26946"];
string[228] scientific = ["1.00e+00", "5e-01", "1.00000e-01", "2.0000000000e-01", "3.000000000e-01", "1.000000000e+23", "1.000000000000e+22", "1.0000000e+21", "1.00000000000e+20", "1.00000000000e-05", "1.00e-06", "1.00000000e-07", "1.230e-04", "4.9406565e-324", "2.225073859e-308", "1.80e+308", "1.234567890e+05", "1.00000000000e+02", "3.14159000e+00", "2.7183e+00", "1.0e+01", "3.750000e-01", "1.2500000000e-01", "2.50e+00", "1.50e+00", "9.990000000000e-01", "9.950000000000e+01", "1.000000000e+15", "1.2346e+17", "1.00000000e+23", "1.152922e+18", "4.503600e+15", "5.000000000000e-02", "2.50e-01", "4.35000000e+00", "1.000000000000e-04", "1.235e+02", "9.007199e+15", "0e+00", "-0.0e+00", "-1.000000e+00", "-5.0e-01", "-1e-01", "-2.00000e-01", "-3.0000000000e-01", "-1.00000e+23", "-1.000000000000e+22", "-1e+21", "-1e+20", "-1.00000000e-05", "-1.0000000e-06", "-1.00000e-07", "-1.2300000000e-04", "-4.9407e-324", "-2.2250739e-308", "-1.79769e+308", "4.9e-324", "4.1445230e-317", "3.4766779e-310", "2.916448808e-303", "2.446494580e-296", "2.052268400649e-289", "1.72e-282", "1.44e-275", "1.211445439e-268", "1.01623409e-261", "8.525e-255", "7.151112e-248", "5.998787256e-241", "5.03215e-234", "4.2212713e-227", "3.541058984e-220", "2.970e-213", "2.4917987378e-206", "2.09027228261e-199", "1.8e-192", "1.47089836e-185", "1.233879e-178", "1e-171", "8.683e-165", "7.2835358703e-158", "6.11e-151", "5e-144", "4.2994407088e-137", "3.60663227e-130", "3e-123", "2.5379418e-116", "2.128979920004e-109", "1.7859177989e-102", "1.498e-95", "1.2567279e-88", "1.05421979432e-81", "8.8434366004e-75", "7e-68", "6.22e-61", "5.2202435744e-54", "4.37905770e-47", "3.673e-40", "3.081487911020e-33", "2.58493941e-26", "2.17e-19", "1.82e-12", "1.5258789062e-05", "1.28e+02", "1.073742e+09", "9.0072e+15", "7.555786e+22", "6.33825300e+29", "5.316912e+36", "4.5e+43", "3.741444e+50", "3.1386e+57", "2.63e+64", "2.20856e+71", "1.85e+78", "1.5541e+85", "1.3037030e+92", "1.0936254e+99", "9.173994463960e+105", "7.695704e+112", "6.4556e+119", "5.4153705e+126", "4.54e+133", "3.81073e+140", "3.19667052e+147", "2.6815616e+154", "2.2494569e+161", "1.9e+168", "1.58291457e+175", "1.327844982042e+182", "1.11387710e+189", "9.343878384890e+195", "7.8382e+202", "6.6e+209", "5.516e+216", "4.627e+223", "4e+230", "3.3e+237", "2.7312187e+244", "2.2911123135e+251", "1.92192e+258", "1.612227e+265", "1.35243400e+272", "1.134503866942e+279", "9.5169082143e+285", "7.98e+292", "7e+299", "5.61779e+306", "1.08e-111", "3.82784e+278", "1.19114783e+91", "3.7276561872e+279", "1.365300791e+237", "5.342958922647e+243", "1.62782053e+75", "2.428e-49", "1.2e+45", "2.606387362e+211", "6.86266287974e+170", "5.2346e-06", "4.72896216e+168", "1.025071816248e-36", "4.241361122e-307", "6.87891541964e-259", "1.429e-131", "3.235378915e-248", "1.4271e-301", "2.258599389e+241", "2.1119e-06", "6.2918453136e-115", "9.12329217e-64", "2.40e-94", "7.9e+307", "8.738245870118e-89", "1.2115e-76", "9.1e+151", "1.624623e+85", "7.19e+137", "2.2e-253", "5.9133302393e+23", "2e-131", "8.517071473104e-222", "3.92193e+89", "5.53595001948e+133", "3.627193966900e-118", "5.53e+214", "2.9386314651e-81", "2.1118269708e-24", "4.4e+01", "8.44480767e+17", "1.26374e+17", "3.6633e+04", "6.371744846e+00", "2.0e+16", "-6.2066e+04", "-9.26078e+06", "-2.49e+15", "7.8159e+11", "2.0047e+03", "-5.0e+17", "6e+08", "2.505284e+10", "2.633131169991e+06", "1.0370e+06", "8.630937e+14", "1e+02", "-7e+15", "1.3e-12", "7.749e+14", "-3e-01", "1.0194e-04", "1.5e+03", "-1.1e+15", "7.1955112414e+15", "-7.44301548e+02", "1.601455927e+02", "-5.09615e+02", "3.77999e+20", "4.314173511243e+19", "2.2581888e+18", "2.135828319e-10", "-1.1381169657e-08", "1.4908423021e-10", "-1.39529202398e+07", "-1.477005795e+08", "-4e-05", "1.239607e+13", "1.57195e+10"];
float32[348] singles = [1.100000023841858, 0.10000000149011612, 0.20000000298023224, 16777216.0, 16777216.0, 3.4028234663852886e+38, 1.401298464324817e-45, 1.1754943508222875e-38, 1.1754942106924411e-38, 2.5, 100.0, 10000000000.0, 9.999999747378752e-06, 123456.7890625, 0.5, 3.141590118408203, 8388608.0, 0.0010000000474974513, 7.000000024071085e-10, 1.0000000200408773e+21, 1.0000000200408773e+20, 1.401298464324817e-45, 5.605193857299268e-45, 2.2420775429197073e-44, 8.96831017167883e-44, 3.587324068671532e-43, 1.4349296274686127e-42, 5.739718509874451e-42, 2.2958874039497803e-41, 9.183549615799121e-41, 3.6734198463196485e-40, 1.4693679385278594e-39, 5.877471754111438e-39, 2.350988701644575e-38, 9.4039548065783e-38, 3.76158192263132e-37, 1.504632769052528e-36, 6.018531076210112e-36, 2.407412430484045e-35, 9.62964972193618e-35, 3.851859888774472e-34, 1.5407439555097887e-33, 6.162975822039155e-33, 2.465190328815662e-32, 9.860761315262648e-32, 3.944304526105059e-31, 1.5777218104420236e-30, 6.310887241768095e-30, 2.524354896707238e-29, 1.0097419586828951e-28, 4.0389678347315804e-28, 1.6155871338926322e-27, 6.462348535570529e-27, 2.5849394142282115e-26, 1.0339757656912846e-25, 4.1359030627651384e-25, 1.6543612251060553e-24, 6.617444900424222e-24, 2.6469779601696886e-23, 1.0587911840678754e-22, 4.235164736271502e-22, 1.6940658945086007e-21, 6.776263578034403e-21, 2.710505431213761e-20, 1.0842021724855044e-19, 4.336808689942018e-19, 1.734723475976807e-18, 6.938893903907228e-18, 2.7755575615628914e-17, 1.1102230246251565e-16, 4.440892098500626e-16, 1.7763568394002505e-15, 7.105427357601002e-15, 2.842170943040401e-14, 1.1368683772161603e-13, 4.547473508864641e-13, 1.8189894035458565e-12, 7.275957614183426e-12, 2.9103830456733704e-11, 1.1641532182693481e-10, 4.656612873077393e-10, 1.862645149230957e-09, 7.450580596923828e-09, 2.9802322387695312e-08, 1.1920928955078125e-07, 4.76837158203125e-07, 1.9073486328125e-06, 7.62939453125e-06, 3.0517578125e-05, 0.0001220703125, 0.00048828125, 0.001953125, 0.0078125, 0.03125, 0.125, 0.5, 2.0, 8.0, 32.0, 128.0, 512.0, 2048.0, 8192.0, 32768.0, 131072.0, 524288.0, 2097152.0, 8388608.0, 33554432.0, 134217728.0, 536870912.0, 2147483648.0, 8589934592.0, 34359738368.0, 137438953472.0, 549755813888.0, 2199023255552.0, 8796093022208.0, 35184372088832.0, 140737488355328.0, 562949953421312.0, 2251799813685248.0, 9007199254740992.0, 3.602879701896397e+16, 1.4411518807585587e+17, 5.764607523034235e+17, 2.305843009213694e+18, 9.223372036854776e+18, 3.6893488147419103e+19, 1.4757395258967641e+20, 5.902958103587057e+20, 2.3611832414348226e+21, 9.44473296573929e+21, 3.777893186295716e+22, 1.5111572745182865e+23, 6.044629098073146e+23, 2.4178516392292583e+24, 9.671406556917033e+24, 3.8685626227668134e+25, 1.5474250491067253e+26, 6.189700196426902e+26, 2.4758800785707605e+27, 9.903520314283042e+27, 3.961408125713217e+28, 1.5845632502852868e+29, 6.338253001141147e+29, 2.535301200456459e+30, 1.0141204801825835e+31, 4.056481920730334e+31, 1.6225927682921336e+32, 6.490371073168535e+32, 2.596148429267414e+33, 1.0384593717069655e+34, 4.153837486827862e+34, 1.661534994731145e+35, 6.64613997892458e+35, 2.658455991569832e+36, 1.0633823966279327e+37, 4.253529586511731e+37, 1.7014118346046923e+38, 427651744.0, 2.547185162393058e-22, 36195364.0, 0.0028192992322146893, 4.149376763962209e-06, 1.757453425343194e+37, 72855085645824.0, 2.9650671891113374e+29, 9.95041109722472e-30, 488259465510912.0, 4.14811129889614e-10, 1.4785936061398485e+28, 2.646033763885498, 202569800482816.0, 7.538920943653317e+36, 9.575886329773249e+32, 1.5220653183555626e-19, 5.945976155885546e-09, 1.9113986669062931e-16, 6.015948379448978e+34, 5.342517195590145e+24, 1.7715294509113022e-11, 1.6885224446437366e-34, 2.903176340503003e+18, 7.812536921489408e+16, 2.755411696853116e-05, 2.9708392432160193e+29, 3.9790334004554565e-37, 2.904117641493638e-17, 2248378335363072.0, 1.8420688422966097e-13, 1.2503969188794946e-16, 6.597132988574597e+17, 4.38898073105801e+35, 1.1230665797270545e+21, 2.659394495080713e+23, 2.369772369240988e+32, 1.656860077416218e-29, 4.51708271210615e-16, 9.330894070583363e+20, 7.910472277217195e-07, 2.321947531979101e-14, 8032745.5, 2.348468243234444e+31, 2.2365304188707697e-37, 3670.32080078125, 3.2204913067510088e+35, 1.5241902791979234e-12, 1.0235620917675525e-23, 103605.0390625, 1.3223037229241542e-34, 2.521476515233792e+17, 2.879321690587757e-30, 1.9435902597540802e-26, 3127584768.0, 1.776406762036087e-29, 8.080686709925441e-30, 4.675178657566546e-36, 2.9650597282768466e-29, 114.01451873779297, 1.1891872903069256e-11, 61064712.0, 511472664576.0, 6.209114899194697e+36, 2072314118144.0, 1.2201263965643493e-15, 1.0316582605282578e-36, 3.3209635148523375e-06, 2098308.75, 3.363048035533054e-29, 1.6424616874899378e-11, 6.276676458583097e-07, 2.4793944675138846e-09, 7.257958678508203e-10, 1.7107772047041593e-20, 4.2724425248715204e+21, 1.690171416157682e-06, 7141617827840.0, 5.119102944322095e-22, 3.2499047330766767e+21, 2.6588124920684174e-29, 8.137728367850519e-25, 4074.146240234375, 9.590404181385469e-37, 5981549887488.0, 1.167541762260992e+17, 6.616844598312106e+33, 3237.216552734375, 292887912775680.0, 34535.0234375, 2.377861285651761e-08, 2.0553641480457314e+30, 2.761294320824517e-19, 2.589513941102656e+22, 1.459639001098759e-27, 2.4284166878836833e-21, 3.7252302225969456e-37, 1.1741075137581145e-37, 1.2967961098719418e-17, 3142.440673828125, 0.04949638247489929, 9.428126759303268e-07, 1.3330700063248929e-18, 4.554634538510147e+35, 1.1188495469731732e-14, 3.021764823444129e-28, 3.5771351967105163e+34, 12499253248.0, 33127126.0, 4.67986929328606e-32, 2.9797892699234503e+26, 3.931947020024772e-25, 25615.099609375, 5.206802147923324e-36, 799667453952.0, 3.4233303549754906e+28, 7.272307084850065e+35, 1.8267215748698192e+25, 25.535350799560547, 9.115362891538693e+24, 1.6454149565885646e-18, 1.111885214975794e-12, 36066648064.0, 79.03518676757812, 8.56060929747326e+19, 1.1359405370763117e+19, 1.8182410939826706e-14, 1.3495665822854665e-34, 3109717803008.0, 3.735405538540504e+26, 4.5157538056905535e-38, 14875072512.0, 1.4897333512436535e+37, 3.140024134160749e-20, 0.011402626521885395, 1.8527549618710677e-34, 3.640932426731568e+36, 7.58390780414503e-22, 130932078018560.0, 9.654279708862305, 1.6830285110172996e+28, 2.6120617463746325e-27, 1.7793753983875485e+33, 1.6319230855404847e+19, 3.950124253898098e+31, 9.20798845580018e+28, 3.4660907033412514e-08, 1.911347634556109e+16, 1.010118697385253e+37, 3.324757830158731e-11, 1.209576654210883e+35, 4.842918837695099e+22, 5.1609991576717656e-21, 115808.625, 4.786896882988296e+25, 1.106794873838104e-17, 1.8045414338155319e-12, 1.4563501974730738e+28, 1.920021021219723e-23, 4.493606597932853e-10, 2.811250364442852e-11, 0.0015356814255937934, 73867812864.0, 0.0011088027385994792, 12219.603515625, 2.0890286262825964e+24, 0.021250629797577858, 8.304138051300919e+30, 3.5682443186021413e+22, 48950916677632.0, 3415008.5, 6.82681111619475e-15, 1.8247317120218636e-24, 9.84423529559585e-27, 15223494656.0, 2.617324241782626e-07, 4.418913753416339e-27, 0.1404142528772354, 4.59450051454919e+30, 1.5984642411119403e-38, -1.100000023841858, -0.10000000149011612, -0.20000000298023224, -16777216.0, -16777216.0, -3.4028234663852886e+38, -1.401298464324817e-45, -1.1754943508222875e-38];
string[348] singleText = ["1.1", "0.1", "0.2", "16777216", "16777216", "3.4028235e+38", "1e-45", "1.1754944e-38", "1.1754942e-38", "2.5", "100", "10000000000", "0.00001", "123456.79", "0.5", "3.14159", "8388608", "0.001", "7e-10", "1e+21", "100000000000000000000", "1e-45", "6e-45", "2.2e-44", "9e-44", "3.59e-43", "1.435e-42", "5.74e-42", "2.2959e-41", "9.1835e-41", "3.67342e-40", "1.469368e-39", "5.877472e-39", "2.3509887e-38", "9.403955e-38", "3.761582e-37", "1.5046328e-36", "6.018531e-36", "2.4074124e-35", "9.62965e-35", "3.85186e-34", "1.540744e-33", "6.162976e-33", "2.4651903e-32", "9.8607613e-32", "3.9443045e-31", "1.5777218e-30", "6.3108872e-30", "2.524355e-29", "1.009742e-28", "4.038968e-28", "1.6155871e-27", "6.4623485e-27", "2.5849394e-26", "1.0339758e-25", "4.135903e-25", "1.6543612e-24", "6.617445e-24", "2.646978e-23", "1.0587912e-22", "4.2351647e-22", "1.6940659e-21", "6.7762636e-21", "2.7105054e-20", "1.0842022e-19", "4.3368087e-19", "1.7347235e-18", "6.938894e-18", "2.7755576e-17", "1.110223e-16", "4.440892e-16", "1.7763568e-15", "7.1054274e-15", "2.842171e-14", "1.1368684e-13", "4.5474735e-13", "1.8189894e-12", "7.275958e-12", "2.910383e-11", "1.1641532e-10", "4.656613e-10", "1.8626451e-09", "7.450581e-09", "2.9802322e-08", "1.1920929e-07", "4.7683716e-07", "1.9073486e-06", "7.6293945e-06", "0.000030517578", "0.00012207031", "0.00048828125", "0.001953125", "0.0078125", "0.03125", "0.125", "0.5", "2", "8", "32", "128", "512", "2048", "8192", "32768", "131072", "524288", "2097152", "8388608", "33554432", "134217730", "536870900", "2147483600", "8589935000", "34359740000", "137438950000", "549755800000", "2199023300000", "8796093000000", "35184372000000", "140737490000000", "562949950000000", "2251799800000000", "9007199000000000", "36028797000000000", "144115190000000000", "576460750000000000", "2305843000000000000", "9223372000000000000", "36893490000000000000", "147573950000000000000", "590295800000000000000", "2.3611832e+21", "9.444733e+21", "3.7778932e+22", "1.5111573e+23", "6.044629e+23", "2.4178516e+24", "9.671407e+24", "3.8685626e+25", "1.5474251e+26", "6.1897002e+26", "2.4758801e+27", "9.9035203e+27", "3.9614081e+28", "1.5845633e+29", "6.338253e+29", "2.5353012e+30", "1.0141205e+31", "4.056482e+31", "1.6225928e+32", "6.490371e+32", "2.5961484e+33", "1.0384594e+34", "4.1538375e+34", "1.661535e+35", "6.64614e+35", "2.658456e+36", "1.0633824e+37", "4.2535296e+37", "1.7014118e+38", "427651740", "2.5471852e-22", "36195364", "0.0028192992", "4.1493768e-06", "1.7574534e+37", "72855086000000", "2.9650672e+29", "9.950411e-30", "488259470000000", "4.1481113e-10", "1.4785936e+28", "2.6460338", "202569800000000", "7.538921e+36", "9.575886e+32", "1.5220653e-19", "5.945976e-09", "1.9113987e-16", "6.0159484e+34", "5.342517e+24", "1.7715295e-11", "1.6885224e-34", "2903176300000000000", "78125370000000000", "0.000027554117", "2.9708392e+29", "3.9790334e-37", "2.9041176e-17", "2248378300000000", "1.8420688e-13", "1.2503969e-16", "659713300000000000", "4.3889807e+35", "1.1230666e+21", "2.6593945e+23", "2.3697724e+32", "1.6568601e-29", "4.5170827e-16", "933089400000000000000", "7.910472e-07", "2.3219475e-14", "8032745.5", "2.3484682e+31", "2.2365304e-37", "3670.3208", "3.2204913e+35", "1.5241903e-12", "1.0235621e-23", "103605.04", "1.3223037e-34", "252147650000000000", "2.8793217e-30", "1.9435903e-26", "3127584800", "1.7764068e-29", "8.080687e-30", "4.6751787e-36", "2.9650597e-29", "114.01452", "1.1891873e-11", "61064710", "511472660000", "6.209115e+36", "2072314100000", "1.2201264e-15", "1.0316583e-36", "3.3209635e-06", "2098308.8", "3.363048e-29", "1.6424617e-11", "6.2766765e-07", "2.4793945e-09", "7.2579587e-10", "1.7107772e-20", "4.2724425e+21", "1.6901714e-06", "7141618000000", "5.119103e-22", "3.2499047e+21", "2.6588125e-29", "8.1377284e-25", "4074.1462", "9.590404e-37", "5981550000000", "116754180000000000", "6.6168446e+33", "3237.2166", "292887900000000", "34535.023", "2.3778613e-08", "2.0553641e+30", "2.7612943e-19", "2.589514e+22", "1.459639e-27", "2.4284167e-21", "3.7252302e-37", "1.1741075e-37", "1.2967961e-17", "3142.4407", "0.049496382", "9.428127e-07", "1.33307e-18", "4.5546345e+35", "1.11884955e-14", "3.0217648e-28", "3.5771352e+34", "12499253000", "33127126", "4.6798693e-32", "2.9797893e+26", "3.931947e-25", "25615.1", "5.206802e-36", "799667450000", "3.4233304e+28", "7.272307e+35", "1.8267216e+25", "25.53535", "9.115363e+24", "1.645415e-18", "1.1118852e-12", "36066650000", "79.03519", "85606090000000000000", "11359405000000000000", "1.8182411e-14", "1.3495666e-34", "3109717800000", "3.7354055e+26", "4.5157538e-38", "14875073000", "1.4897334e+37", "3.140024e-20", "0.0114026265", "1.852755e-34", "3.6409324e+36", "7.583908e-22", "130932080000000", "9.65428", "1.6830285e+28", "2.6120617e-27", "1.7793754e+33", "16319231000000000000", "3.9501243e+31", "9.207988e+28", "3.4660907e-08", "19113476000000000", "1.0101187e+37", "3.324758e-11", "1.2095767e+35", "4.842919e+22", "5.160999e-21", "115808.625", "4.786897e+25", "1.1067949e-17", "1.8045414e-12", "1.4563502e+28", "1.920021e-23", "4.4936066e-10", "2.8112504e-11", "0.0015356814", "73867810000", "0.0011088027", "12219.604", "2.0890286e+24", "0.02125063", "8.304138e+30", "3.5682443e+22", "48950917000000", "3415008.5", "6.826811e-15", "1.8247317e-24", "9.844235e-27", "15223495000", "2.6173242e-07", "4.4189138e-27", "0.14041425", "4.5945005e+30", "1.5984642e-38", "-1.1", "-0.1", "-0.2", "-16777216", "-16777216", "-3.4028235e+38", "-1e-45", "-1.1754944e-38"];

func differ(string got, string want) int {
    return got == want ? 0 : 1;
}

func main() {
    char[1500] buf;
    int shortBad;
    int fixedBad;
    int expBad;
    int singleBad;
    for i in 0..227 {
        shortBad += differ(conv.ftoa(buf[:], xs[i]) ?? "error", shortest[i]);
        fixedBad += differ(conv.formatFixed(buf[:], xs[i], ns[i]) ?? "error", fixed[i]);
        expBad += differ(conv.formatExp(buf[:], xs[i], ns[i], false) ?? "error", scientific[i]);
    }
    for i in 0..347 {
        singleBad += differ(conv.ftoa32(buf[:], singles[i]) ?? "error", singleText[i]);
    }
    check(shortBad == 0, "ftoa writes the shortest text that reads back");
    check(fixedBad == 0, "formatFixed matches the exact decimal expansion");
    check(expBad == 0, "formatExp matches the exact decimal expansion");
    check(singleBad == 0, "ftoa32 writes the shortest text of a float32");

    int special;
    special += differ(conv.ftoa(buf[:], math.nan()) ?? "error", "NaN");
    special += differ(conv.ftoa(buf[:], math.inf(1)) ?? "error", "inf");
    special += differ(conv.formatFixed(buf[:], math.inf(-1), 2) ?? "error", "-inf");
    special += differ(conv.formatExp(buf[:], math.nan(), 2, true) ?? "error", "NaN");
    special += differ(conv.formatExp(buf[:], 1234.5, 2, true) ?? "error", "1.23E+03");
    special += differ(conv.formatFixed(buf[:], -0.001, 2) ?? "error", "-0.00");
    check(special == 0, "NaN, the infinities and signed zeros keep their text");

    char[3] tiny;
    check((conv.ftoa(tiny[:], 123456.0) ?? "no space") == "no space", "a buffer that is too small is reported");
}
`, expected: []string{"ftoa writes the shortest text that reads back", "formatFixed matches the exact decimal expansion", "formatExp matches the exact decimal expansion", "ftoa32 writes the shortest text of a float32", "NaN, the infinities and signed zeros keep their text", "a buffer that is too small is reported"}},

	{dir: "fmt_float", src: `import "std/fmt";
import "std/math";

func render(char[] buf, string format, ...args) string {
    let n, _ = fmt.sprintf(buf, format, args...);
    return string(buf[:n]);
}

func differ(string got, string want) int {
    return got == want ? 0 : 1;
}

float32[8] singles = [1.100000023841858, 0.10000000149011612, 2.5, 16777216.0, 3.4028234663852886e+38, 1.401298464324817e-45, 0.30000001192092896, 100.0];

func main() {
    char[1400] buf;
    int bad;
    int single;
    bad += differ(render(buf[:], "[{}]", 1.0), "[1]");
    bad += differ(render(buf[:], "[{}]", -1.0), "[-1]");
    bad += differ(render(buf[:], "[{}]", 0.5), "[0.5]");
    bad += differ(render(buf[:], "[{}]", 0.1), "[0.1]");
    bad += differ(render(buf[:], "[{}]", -0.1), "[-0.1]");
    bad += differ(render(buf[:], "[{}]", 123456.789), "[123456.789]");
    bad += differ(render(buf[:], "[{}]", -2.5), "[-2.5]");
    bad += differ(render(buf[:], "[{}]", 1e+21), "[1e+21]");
    bad += differ(render(buf[:], "[{}]", 1e-07), "[1e-07]");
    bad += differ(render(buf[:], "[{}]", 5e-324), "[5e-324]");
    bad += differ(render(buf[:], "[{}]", 1.7976931348623157e+308), "[1.7976931348623157e+308]");
    bad += differ(render(buf[:], "[{}]", 0.0), "[0]");
    bad += differ(render(buf[:], "[{}]", -0.0), "[-0]");
    bad += differ(render(buf[:], "[{}]", 100.0), "[100]");
    bad += differ(render(buf[:], "[{}]", 0.000123), "[0.000123]");
    bad += differ(render(buf[:], "[{}]", 3.14159), "[3.14159]");
    bad += differ(render(buf[:], "[{}]", 9.995), "[9.995]");
    bad += differ(render(buf[:], "[{}]", 0.375), "[0.375]");
    bad += differ(render(buf[:], "[{}]", 1e+20), "[100000000000000000000]");
    bad += differ(render(buf[:], "[{}]", 1e-05), "[0.00001]");
    bad += differ(render(buf[:], "[{}]", 99.5), "[99.5]");
    bad += differ(render(buf[:], "[{}]", math.nan()), "[NaN]");
    bad += differ(render(buf[:], "[{}]", math.inf(1)), "[inf]");
    bad += differ(render(buf[:], "[{}]", math.inf(-1)), "[-inf]");
    bad += differ(render(buf[:], "[{.3}]", 1.0), "[1.000]");
    bad += differ(render(buf[:], "[{.3}]", -1.0), "[-1.000]");
    bad += differ(render(buf[:], "[{.3}]", 0.5), "[0.500]");
    bad += differ(render(buf[:], "[{.3}]", 0.1), "[0.100]");
    bad += differ(render(buf[:], "[{.3}]", -0.1), "[-0.100]");
    bad += differ(render(buf[:], "[{.3}]", 123456.789), "[123456.789]");
    bad += differ(render(buf[:], "[{.3}]", -2.5), "[-2.500]");
    bad += differ(render(buf[:], "[{.3}]", 1e+21), "[1000000000000000000000.000]");
    bad += differ(render(buf[:], "[{.3}]", 1e-07), "[0.000]");
    bad += differ(render(buf[:], "[{.3}]", 5e-324), "[0.000]");
    bad += differ(render(buf[:], "[{.3}]", 1.7976931348623157e+308), "[179769313486231570814527423731704356798070567525844996598917476803157260780028538760589558632766878171540458953514382464234321326889464182768467546703537516986049910576551282076245490090389328944075868508455133942304583236903222948165808559332123348274797826204144723168738177180919299881250404026184124858368.000]");
    bad += differ(render(buf[:], "[{.3}]", 0.0), "[0.000]");
    bad += differ(render(buf[:], "[{.3}]", -0.0), "[-0.000]");
    bad += differ(render(buf[:], "[{.3}]", 100.0), "[100.000]");
    bad += differ(render(buf[:], "[{.3}]", 0.000123), "[0.000]");
    bad += differ(render(buf[:], "[{.3}]", 3.14159), "[3.142]");
    bad += differ(render(buf[:], "[{.3}]", 9.995), "[9.995]");
    bad += differ(render(buf[:], "[{.3}]", 0.375), "[0.375]");
    bad += differ(render(buf[:], "[{.3}]", 1e+20), "[100000000000000000000.000]");
    bad += differ(render(buf[:], "[{.3}]", 1e-05), "[0.000]");
    bad += differ(render(buf[:], "[{.3}]", 99.5), "[99.500]");
    bad += differ(render(buf[:], "[{.3}]", math.nan()), "[NaN]");
    bad += differ(render(buf[:], "[{.3}]", math.inf(1)), "[inf]");
    bad += differ(render(buf[:], "[{.3}]", math.inf(-1)), "[-inf]");
    bad += differ(render(buf[:], "[{f}]", 1.0), "[1.000000]");
    bad += differ(render(buf[:], "[{f}]", -1.0), "[-1.000000]");
    bad += differ(render(buf[:], "[{f}]", 0.5), "[0.500000]");
    bad += differ(render(buf[:], "[{f}]", 0.1), "[0.100000]");
    bad += differ(render(buf[:], "[{f}]", -0.1), "[-0.100000]");
    bad += differ(render(buf[:], "[{f}]", 123456.789), "[123456.789000]");
    bad += differ(render(buf[:], "[{f}]", -2.5), "[-2.500000]");
    bad += differ(render(buf[:], "[{f}]", 1e+21), "[1000000000000000000000.000000]");
    bad += differ(render(buf[:], "[{f}]", 1e-07), "[0.000000]");
    bad += differ(render(buf[:], "[{f}]", 5e-324), "[0.000000]");
    bad += differ(render(buf[:], "[{f}]", 1.7976931348623157e+308), "[179769313486231570814527423731704356798070567525844996598917476803157260780028538760589558632766878171540458953514382464234321326889464182768467546703537516986049910576551282076245490090389328944075868508455133942304583236903222948165808559332123348274797826204144723168738177180919299881250404026184124858368.000000]");
    bad += differ(render(buf[:], "[{f}]", 0.0), "[0.000000]");
    bad += differ(render(buf[:], "[{f}]", -0.0), "[-0.000000]");
    bad += differ(render(buf[:], "[{f}]", 100.0), "[100.000000]");
    bad += differ(render(buf[:], "[{f}]", 0.000123), "[0.000123]");
    bad += differ(render(buf[:], "[{f}]", 3.14159), "[3.141590]");
    bad += differ(render(buf[:], "[{f}]", 9.995), "[9.995000]");
    bad += differ(render(buf[:], "[{f}]", 0.375), "[0.375000]");
    bad += differ(render(buf[:], "[{f}]", 1e+20), "[100000000000000000000.000000]");
    bad += differ(render(buf[:], "[{f}]", 1e-05), "[0.000010]");
    bad += differ(render(buf[:], "[{f}]", 99.5), "[99.500000]");
    bad += differ(render(buf[:], "[{f}]", math.nan()), "[NaN]");
    bad += differ(render(buf[:], "[{f}]", math.inf(1)), "[inf]");
    bad += differ(render(buf[:], "[{f}]", math.inf(-1)), "[-inf]");
    bad += differ(render(buf[:], "[{.2f}]", 1.0), "[1.00]");
    bad += differ(render(buf[:], "[{.2f}]", -1.0), "[-1.00]");
    bad += differ(render(buf[:], "[{.2f}]", 0.5), "[0.50]");
    bad += differ(render(buf[:], "[{.2f}]", 0.1), "[0.10]");
    bad += differ(render(buf[:], "[{.2f}]", -0.1), "[-0.10]");
    bad += differ(render(buf[:], "[{.2f}]", 123456.789), "[123456.79]");
    bad += differ(render(buf[:], "[{.2f}]", -2.5), "[-2.50]");
    bad += differ(render(buf[:], "[{.2f}]", 1e+21), "[1000000000000000000000.00]");
    bad += differ(render(buf[:], "[{.2f}]", 1e-07), "[0.00]");
    bad += differ(render(buf[:], "[{.2f}]", 5e-324), "[0.00]");
    bad += differ(render(buf[:], "[{.2f}]", 1.7976931348623157e+308), "[179769313486231570814527423731704356798070567525844996598917476803157260780028538760589558632766878171540458953514382464234321326889464182768467546703537516986049910576551282076245490090389328944075868508455133942304583236903222948165808559332123348274797826204144723168738177180919299881250404026184124858368.00]");
    bad += differ(render(buf[:], "[{.2f}]", 0.0), "[0.00]");
    bad += differ(render(buf[:], "[{.2f}]", -0.0), "[-0.00]");
    bad += differ(render(buf[:], "[{.2f}]", 100.0), "[100.00]");
    bad += differ(render(buf[:], "[{.2f}]", 0.000123), "[0.00]");
    bad += differ(render(buf[:], "[{.2f}]", 3.14159), "[3.14]");
    bad += differ(render(buf[:], "[{.2f}]", 9.995), "[9.99]");
    bad += differ(render(buf[:], "[{.2f}]", 0.375), "[0.38]");
    bad += differ(render(buf[:], "[{.2f}]", 1e+20), "[100000000000000000000.00]");
    bad += differ(render(buf[:], "[{.2f}]", 1e-05), "[0.00]");
    bad += differ(render(buf[:], "[{.2f}]", 99.5), "[99.50]");
    bad += differ(render(buf[:], "[{.2f}]", math.nan()), "[NaN]");
    bad += differ(render(buf[:], "[{.2f}]", math.inf(1)), "[inf]");
    bad += differ(render(buf[:], "[{.2f}]", math.inf(-1)), "[-inf]");
    bad += differ(render(buf[:], "[{e}]", 1.0), "[1.000000e+00]");
    bad += differ(render(buf[:], "[{e}]", -1.0), "[-1.000000e+00]");
    bad += differ(render(buf[:], "[{e}]", 0.5), "[5.000000e-01]");
    bad += differ(render(buf[:], "[{e}]", 0.1), "[1.000000e-01]");
    bad += differ(render(buf[:], "[{e}]", -0.1), "[-1.000000e-01]");
    bad += differ(render(buf[:], "[{e}]", 123456.789), "[1.234568e+05]");
    bad += differ(render(buf[:], "[{e}]", -2.5), "[-2.500000e+00]");
    bad += differ(render(buf[:], "[{e}]", 1e+21), "[1.000000e+21]");
    bad += differ(render(buf[:], "[{e}]", 1e-07), "[1.000000e-07]");
    bad += differ(render(buf[:], "[{e}]", 5e-324), "[4.940656e-324]");
    bad += differ(render(buf[:], "[{e}]", 1.7976931348623157e+308), "[1.797693e+308]");
    bad += differ(render(buf[:], "[{e}]", 0.0), "[0.000000e+00]");
    bad += differ(render(buf[:], "[{e}]", -0.0), "[-0.000000e+00]");
    bad += differ(render(buf[:], "[{e}]", 100.0), "[1.000000e+02]");
    bad += differ(render(buf[:], "[{e}]", 0.000123), "[1.230000e-04]");
    bad += differ(render(buf[:], "[{e}]", 3.14159), "[3.141590e+00]");
    bad += differ(render(buf[:], "[{e}]", 9.995), "[9.995000e+00]");
    bad += differ(render(buf[:], "[{e}]", 0.375), "[3.750000e-01]");
    bad += differ(render(buf[:], "[{e}]", 1e+20), "[1.000000e+20]");
    bad += differ(render(buf[:], "[{e}]", 1e-05), "[1.000000e-05]");
    bad += differ(render(buf[:], "[{e}]", 99.5), "[9.950000e+01]");
    bad += differ(render(buf[:], "[{e}]", math.nan()), "[NaN]");
    bad += differ(render(buf[:], "[{e}]", math.inf(1)), "[inf]");
    bad += differ(render(buf[:], "[{e}]", math.inf(-1)), "[-inf]");
    bad += differ(render(buf[:], "[{.3e}]", 1.0), "[1.000e+00]");
    bad += differ(render(buf[:], "[{.3e}]", -1.0), "[-1.000e+00]");
    bad += differ(render(buf[:], "[{.3e}]", 0.5), "[5.000e-01]");
    bad += differ(render(buf[:], "[{.3e}]", 0.1), "[1.000e-01]");
    bad += differ(render(buf[:], "[{.3e}]", -0.1), "[-1.000e-01]");
    bad += differ(render(buf[:], "[{.3e}]", 123456.789), "[1.235e+05]");
    bad += differ(render(buf[:], "[{.3e}]", -2.5), "[-2.500e+00]");
    bad += differ(render(buf[:], "[{.3e}]", 1e+21), "[1.000e+21]");
    bad += differ(render(buf[:], "[{.3e}]", 1e-07), "[1.000e-07]");
    bad += differ(render(buf[:], "[{.3e}]", 5e-324), "[4.941e-324]");
    bad += differ(render(buf[:], "[{.3e}]", 1.7976931348623157e+308), "[1.798e+308]");
    bad += differ(render(buf[:], "[{.3e}]", 0.0), "[0.000e+00]");
    bad += differ(render(buf[:], "[{.3e}]", -0.0), "[-0.000e+00]");
    bad += differ(render(buf[:], "[{.3e}]", 100.0), "[1.000e+02]");
    bad += differ(render(buf[:], "[{.3e}]", 0.000123), "[1.230e-04]");
    bad += differ(render(buf[:], "[{.3e}]", 3.14159), "[3.142e+00]");
    bad += differ(render(buf[:], "[{.3e}]", 9.995), "[9.995e+00]");
    bad += differ(render(buf[:], "[{.3e}]", 0.375), "[3.750e-01]");
    bad += differ(render(buf[:], "[{.3e}]", 1e+20), "[1.000e+20]");
    bad += differ(render(buf[:], "[{.3e}]", 1e-05), "[1.000e-05]");
    bad += differ(render(buf[:], "[{.3e}]", 99.5), "[9.950e+01]");
    bad += differ(render(buf[:], "[{.3e}]", math.nan()), "[NaN]");
    bad += differ(render(buf[:], "[{.3e}]", math.inf(1)), "[inf]");
    bad += differ(render(buf[:], "[{.3e}]", math.inf(-1)), "[-inf]");
    bad += differ(render(buf[:], "[{.2E}]", 1.0), "[1.00E+00]");
    bad += differ(render(buf[:], "[{.2E}]", -1.0), "[-1.00E+00]");
    bad += differ(render(buf[:], "[{.2E}]", 0.5), "[5.00E-01]");
    bad += differ(render(buf[:], "[{.2E}]", 0.1), "[1.00E-01]");
    bad += differ(render(buf[:], "[{.2E}]", -0.1), "[-1.00E-01]");
    bad += differ(render(buf[:], "[{.2E}]", 123456.789), "[1.23E+05]");
    bad += differ(render(buf[:], "[{.2E}]", -2.5), "[-2.50E+00]");
    bad += differ(render(buf[:], "[{.2E}]", 1e+21), "[1.00E+21]");
    bad += differ(render(buf[:], "[{.2E}]", 1e-07), "[1.00E-07]");
    bad += differ(render(buf[:], "[{.2E}]", 5e-324), "[4.94E-324]");
    bad += differ(render(buf[:], "[{.2E}]", 1.7976931348623157e+308), "[1.80E+308]");
    bad += differ(render(buf[:], "[{.2E}]", 0.0), "[0.00E+00]");
    bad += differ(render(buf[:], "[{.2E}]", -0.0), "[-0.00E+00]");
    bad += differ(render(buf[:], "[{.2E}]", 100.0), "[1.00E+02]");
    bad += differ(render(buf[:], "[{.2E}]", 0.000123), "[1.23E-04]");
    bad += differ(render(buf[:], "[{.2E}]", 3.14159), "[3.14E+00]");
    bad += differ(render(buf[:], "[{.2E}]", 9.995), "[9.99E+00]");
    bad += differ(render(buf[:], "[{.2E}]", 0.375), "[3.75E-01]");
    bad += differ(render(buf[:], "[{.2E}]", 1e+20), "[1.00E+20]");
    bad += differ(render(buf[:], "[{.2E}]", 1e-05), "[1.00E-05]");
    bad += differ(render(buf[:], "[{.2E}]", 99.5), "[9.95E+01]");
    bad += differ(render(buf[:], "[{.2E}]", math.nan()), "[NaN]");
    bad += differ(render(buf[:], "[{.2E}]", math.inf(1)), "[inf]");
    bad += differ(render(buf[:], "[{.2E}]", math.inf(-1)), "[-inf]");
    bad += differ(render(buf[:], "[{10}]", 1.0), "[         1]");
    bad += differ(render(buf[:], "[{10}]", -1.0), "[        -1]");
    bad += differ(render(buf[:], "[{10}]", 0.5), "[       0.5]");
    bad += differ(render(buf[:], "[{10}]", 0.1), "[       0.1]");
    bad += differ(render(buf[:], "[{10}]", -0.1), "[      -0.1]");
    bad += differ(render(buf[:], "[{10}]", 123456.789), "[123456.789]");
    bad += differ(render(buf[:], "[{10}]", -2.5), "[      -2.5]");
    bad += differ(render(buf[:], "[{10}]", 1e+21), "[     1e+21]");
    bad += differ(render(buf[:], "[{10}]", 1e-07), "[     1e-07]");
    bad += differ(render(buf[:], "[{10}]", 5e-324), "[    5e-324]");
    bad += differ(render(buf[:], "[{10}]", 1.7976931348623157e+308), "[1.7976931348623157e+308]");
    bad += differ(render(buf[:], "[{10}]", 0.0), "[         0]");
    bad += differ(render(buf[:], "[{10}]", -0.0), "[        -0]");
    bad += differ(render(buf[:], "[{10}]", 100.0), "[       100]");
    bad += differ(render(buf[:], "[{10}]", 0.000123), "[  0.000123]");
    bad += differ(render(buf[:], "[{10}]", 3.14159), "[   3.14159]");
    bad += differ(render(buf[:], "[{10}]", 9.995), "[     9.995]");
    bad += differ(render(buf[:], "[{10}]", 0.375), "[     0.375]");
    bad += differ(render(buf[:], "[{10}]", 1e+20), "[100000000000000000000]");
    bad += differ(render(buf[:], "[{10}]", 1e-05), "[   0.00001]");
    bad += differ(render(buf[:], "[{10}]", 99.5), "[      99.5]");
    bad += differ(render(buf[:], "[{10}]", math.nan()), "[       NaN]");
    bad += differ(render(buf[:], "[{10}]", math.inf(1)), "[       inf]");
    bad += differ(render(buf[:], "[{10}]", math.inf(-1)), "[      -inf]");
    bad += differ(render(buf[:], "[{<10}]", 1.0), "[1         ]");
    bad += differ(render(buf[:], "[{<10}]", -1.0), "[-1        ]");
    bad += differ(render(buf[:], "[{<10}]", 0.5), "[0.5       ]");
    bad += differ(render(buf[:], "[{<10}]", 0.1), "[0.1       ]");
    bad += differ(render(buf[:], "[{<10}]", -0.1), "[-0.1      ]");
    bad += differ(render(buf[:], "[{<10}]", 123456.789), "[123456.789]");
    bad += differ(render(buf[:], "[{<10}]", -2.5), "[-2.5      ]");
    bad += differ(render(buf[:], "[{<10}]", 1e+21), "[1e+21     ]");
    bad += differ(render(buf[:], "[{<10}]", 1e-07), "[1e-07     ]");
    bad += differ(render(buf[:], "[{<10}]", 5e-324), "[5e-324    ]");
    bad += differ(render(buf[:], "[{<10}]", 1.7976931348623157e+308), "[1.7976931348623157e+308]");
    bad += differ(render(buf[:], "[{<10}]", 0.0), "[0         ]");
    bad += differ(render(buf[:], "[{<10}]", -0.0), "[-0        ]");
    bad += differ(render(buf[:], "[{<10}]", 100.0), "[100       ]");
    bad += differ(render(buf[:], "[{<10}]", 0.000123), "[0.000123  ]");
    bad += differ(render(buf[:], "[{<10}]", 3.14159), "[3.14159   ]");
    bad += differ(render(buf[:], "[{<10}]", 9.995), "[9.995     ]");
    bad += differ(render(buf[:], "[{<10}]", 0.375), "[0.375     ]");
    bad += differ(render(buf[:], "[{<10}]", 1e+20), "[100000000000000000000]");
    bad += differ(render(buf[:], "[{<10}]", 1e-05), "[0.00001   ]");
    bad += differ(render(buf[:], "[{<10}]", 99.5), "[99.5      ]");
    bad += differ(render(buf[:], "[{<10}]", math.nan()), "[NaN       ]");
    bad += differ(render(buf[:], "[{<10}]", math.inf(1)), "[inf       ]");
    bad += differ(render(buf[:], "[{<10}]", math.inf(-1)), "[-inf      ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 1.0), "[    1.00    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", -1.0), "[   -1.00    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 0.5), "[    0.50    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 0.1), "[    0.10    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", -0.1), "[   -0.10    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 123456.789), "[ 123456.79  ]");
    bad += differ(render(buf[:], "[{^12.2f}]", -2.5), "[   -2.50    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 1e+21), "[1000000000000000000000.00]");
    bad += differ(render(buf[:], "[{^12.2f}]", 1e-07), "[    0.00    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 5e-324), "[    0.00    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 1.7976931348623157e+308), "[179769313486231570814527423731704356798070567525844996598917476803157260780028538760589558632766878171540458953514382464234321326889464182768467546703537516986049910576551282076245490090389328944075868508455133942304583236903222948165808559332123348274797826204144723168738177180919299881250404026184124858368.00]");
    bad += differ(render(buf[:], "[{^12.2f}]", 0.0), "[    0.00    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", -0.0), "[   -0.00    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 100.0), "[   100.00   ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 0.000123), "[    0.00    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 3.14159), "[    3.14    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 9.995), "[    9.99    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 0.375), "[    0.38    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 1e+20), "[100000000000000000000.00]");
    bad += differ(render(buf[:], "[{^12.2f}]", 1e-05), "[    0.00    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", 99.5), "[   99.50    ]");
    bad += differ(render(buf[:], "[{^12.2f}]", math.nan()), "[    NaN     ]");
    bad += differ(render(buf[:], "[{^12.2f}]", math.inf(1)), "[    inf     ]");
    bad += differ(render(buf[:], "[{^12.2f}]", math.inf(-1)), "[    -inf    ]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 1.0), "[*******1.000]");
    bad += differ(render(buf[:], "[{*>12.3f}]", -1.0), "[******-1.000]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 0.5), "[*******0.500]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 0.1), "[*******0.100]");
    bad += differ(render(buf[:], "[{*>12.3f}]", -0.1), "[******-0.100]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 123456.789), "[**123456.789]");
    bad += differ(render(buf[:], "[{*>12.3f}]", -2.5), "[******-2.500]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 1e+21), "[1000000000000000000000.000]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 1e-07), "[*******0.000]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 5e-324), "[*******0.000]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 1.7976931348623157e+308), "[179769313486231570814527423731704356798070567525844996598917476803157260780028538760589558632766878171540458953514382464234321326889464182768467546703537516986049910576551282076245490090389328944075868508455133942304583236903222948165808559332123348274797826204144723168738177180919299881250404026184124858368.000]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 0.0), "[*******0.000]");
    bad += differ(render(buf[:], "[{*>12.3f}]", -0.0), "[******-0.000]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 100.0), "[*****100.000]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 0.000123), "[*******0.000]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 3.14159), "[*******3.142]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 9.995), "[*******9.995]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 0.375), "[*******0.375]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 1e+20), "[100000000000000000000.000]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 1e-05), "[*******0.000]");
    bad += differ(render(buf[:], "[{*>12.3f}]", 99.5), "[******99.500]");
    bad += differ(render(buf[:], "[{*>12.3f}]", math.nan()), "[*********NaN]");
    bad += differ(render(buf[:], "[{*>12.3f}]", math.inf(1)), "[*********inf]");
    bad += differ(render(buf[:], "[{*>12.3f}]", math.inf(-1)), "[********-inf]");
    bad += differ(render(buf[:], "[{010.3f}]", 1.0), "[000001.000]");
    bad += differ(render(buf[:], "[{010.3f}]", -1.0), "[-00001.000]");
    bad += differ(render(buf[:], "[{010.3f}]", 0.5), "[000000.500]");
    bad += differ(render(buf[:], "[{010.3f}]", 0.1), "[000000.100]");
    bad += differ(render(buf[:], "[{010.3f}]", -0.1), "[-00000.100]");
    bad += differ(render(buf[:], "[{010.3f}]", 123456.789), "[123456.789]");
    bad += differ(render(buf[:], "[{010.3f}]", -2.5), "[-00002.500]");
    bad += differ(render(buf[:], "[{010.3f}]", 1e+21), "[1000000000000000000000.000]");
    bad += differ(render(buf[:], "[{010.3f}]", 1e-07), "[000000.000]");
    bad += differ(render(buf[:], "[{010.3f}]", 5e-324), "[000000.000]");
    bad += differ(render(buf[:], "[{010.3f}]", 1.7976931348623157e+308), "[179769313486231570814527423731704356798070567525844996598917476803157260780028538760589558632766878171540458953514382464234321326889464182768467546703537516986049910576551282076245490090389328944075868508455133942304583236903222948165808559332123348274797826204144723168738177180919299881250404026184124858368.000]");
    bad += differ(render(buf[:], "[{010.3f}]", 0.0), "[000000.000]");
    bad += differ(render(buf[:], "[{010.3f}]", -0.0), "[-00000.000]");
    bad += differ(render(buf[:], "[{010.3f}]", 100.0), "[000100.000]");
    bad += differ(render(buf[:], "[{010.3f}]", 0.000123), "[000000.000]");
    bad += differ(render(buf[:], "[{010.3f}]", 3.14159), "[000003.142]");
    bad += differ(render(buf[:], "[{010.3f}]", 9.995), "[000009.995]");
    bad += differ(render(buf[:], "[{010.3f}]", 0.375), "[000000.375]");
    bad += differ(render(buf[:], "[{010.3f}]", 1e+20), "[100000000000000000000.000]");
    bad += differ(render(buf[:], "[{010.3f}]", 1e-05), "[000000.000]");
    bad += differ(render(buf[:], "[{010.3f}]", 99.5), "[000099.500]");
    bad += differ(render(buf[:], "[{010.3f}]", math.nan()), "[       NaN]");
    bad += differ(render(buf[:], "[{010.3f}]", math.inf(1)), "[       inf]");
    bad += differ(render(buf[:], "[{010.3f}]", math.inf(-1)), "[      -inf]");
    bad += differ(render(buf[:], "[{010}]", 1.0), "[0000000001]");
    bad += differ(render(buf[:], "[{010}]", -1.0), "[-000000001]");
    bad += differ(render(buf[:], "[{010}]", 0.5), "[00000000.5]");
    bad += differ(render(buf[:], "[{010}]", 0.1), "[00000000.1]");
    bad += differ(render(buf[:], "[{010}]", -0.1), "[-0000000.1]");
    bad += differ(render(buf[:], "[{010}]", 123456.789), "[123456.789]");
    bad += differ(render(buf[:], "[{010}]", -2.5), "[-0000002.5]");
    bad += differ(render(buf[:], "[{010}]", 1e+21), "[000001e+21]");
    bad += differ(render(buf[:], "[{010}]", 1e-07), "[000001e-07]");
    bad += differ(render(buf[:], "[{010}]", 5e-324), "[00005e-324]");
    bad += differ(render(buf[:], "[{010}]", 1.7976931348623157e+308), "[1.7976931348623157e+308]");
    bad += differ(render(buf[:], "[{010}]", 0.0), "[0000000000]");
    bad += differ(render(buf[:], "[{010}]", -0.0), "[-000000000]");
    bad += differ(render(buf[:], "[{010}]", 100.0), "[0000000100]");
    bad += differ(render(buf[:], "[{010}]", 0.000123), "[000.000123]");
    bad += differ(render(buf[:], "[{010}]", 3.14159), "[0003.14159]");
    bad += differ(render(buf[:], "[{010}]", 9.995), "[000009.995]");
    bad += differ(render(buf[:], "[{010}]", 0.375), "[000000.375]");
    bad += differ(render(buf[:], "[{010}]", 1e+20), "[100000000000000000000]");
    bad += differ(render(buf[:], "[{010}]", 1e-05), "[0000.00001]");
    bad += differ(render(buf[:], "[{010}]", 99.5), "[00000099.5]");
    bad += differ(render(buf[:], "[{010}]", math.nan()), "[       NaN]");
    bad += differ(render(buf[:], "[{010}]", math.inf(1)), "[       inf]");
    bad += differ(render(buf[:], "[{010}]", math.inf(-1)), "[      -inf]");
    bad += differ(render(buf[:], "[{014.2e}]", 1.0), "[0000001.00e+00]");
    bad += differ(render(buf[:], "[{014.2e}]", -1.0), "[-000001.00e+00]");
    bad += differ(render(buf[:], "[{014.2e}]", 0.5), "[0000005.00e-01]");
    bad += differ(render(buf[:], "[{014.2e}]", 0.1), "[0000001.00e-01]");
    bad += differ(render(buf[:], "[{014.2e}]", -0.1), "[-000001.00e-01]");
    bad += differ(render(buf[:], "[{014.2e}]", 123456.789), "[0000001.23e+05]");
    bad += differ(render(buf[:], "[{014.2e}]", -2.5), "[-000002.50e+00]");
    bad += differ(render(buf[:], "[{014.2e}]", 1e+21), "[0000001.00e+21]");
    bad += differ(render(buf[:], "[{014.2e}]", 1e-07), "[0000001.00e-07]");
    bad += differ(render(buf[:], "[{014.2e}]", 5e-324), "[000004.94e-324]");
    bad += differ(render(buf[:], "[{014.2e}]", 1.7976931348623157e+308), "[000001.80e+308]");
    bad += differ(render(buf[:], "[{014.2e}]", 0.0), "[0000000.00e+00]");
    bad += differ(render(buf[:], "[{014.2e}]", -0.0), "[-000000.00e+00]");
    bad += differ(render(buf[:], "[{014.2e}]", 100.0), "[0000001.00e+02]");
    bad += differ(render(buf[:], "[{014.2e}]", 0.000123), "[0000001.23e-04]");
    bad += differ(render(buf[:], "[{014.2e}]", 3.14159), "[0000003.14e+00]");
    bad += differ(render(buf[:], "[{014.2e}]", 9.995), "[0000009.99e+00]");
    bad += differ(render(buf[:], "[{014.2e}]", 0.375), "[0000003.75e-01]");
    bad += differ(render(buf[:], "[{014.2e}]", 1e+20), "[0000001.00e+20]");
    bad += differ(render(buf[:], "[{014.2e}]", 1e-05), "[0000001.00e-05]");
    bad += differ(render(buf[:], "[{014.2e}]", 99.5), "[0000009.95e+01]");
    bad += differ(render(buf[:], "[{014.2e}]", math.nan()), "[           NaN]");
    bad += differ(render(buf[:], "[{014.2e}]", math.inf(1)), "[           inf]");
    bad += differ(render(buf[:], "[{014.2e}]", math.inf(-1)), "[          -inf]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 1.0), "[1.0------]");
    bad += differ(render(buf[:], "[{-<9.1f}]", -1.0), "[-1.0-----]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 0.5), "[0.5------]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 0.1), "[0.1------]");
    bad += differ(render(buf[:], "[{-<9.1f}]", -0.1), "[-0.1-----]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 123456.789), "[123456.8-]");
    bad += differ(render(buf[:], "[{-<9.1f}]", -2.5), "[-2.5-----]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 1e+21), "[1000000000000000000000.0]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 1e-07), "[0.0------]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 5e-324), "[0.0------]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 1.7976931348623157e+308), "[179769313486231570814527423731704356798070567525844996598917476803157260780028538760589558632766878171540458953514382464234321326889464182768467546703537516986049910576551282076245490090389328944075868508455133942304583236903222948165808559332123348274797826204144723168738177180919299881250404026184124858368.0]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 0.0), "[0.0------]");
    bad += differ(render(buf[:], "[{-<9.1f}]", -0.0), "[-0.0-----]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 100.0), "[100.0----]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 0.000123), "[0.0------]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 3.14159), "[3.1------]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 9.995), "[10.0-----]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 0.375), "[0.4------]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 1e+20), "[100000000000000000000.0]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 1e-05), "[0.0------]");
    bad += differ(render(buf[:], "[{-<9.1f}]", 99.5), "[99.5-----]");
    bad += differ(render(buf[:], "[{-<9.1f}]", math.nan()), "[NaN------]");
    bad += differ(render(buf[:], "[{-<9.1f}]", math.inf(1)), "[inf------]");
    bad += differ(render(buf[:], "[{-<9.1f}]", math.inf(-1)), "[-inf-----]");
    bad += differ(render(buf[:], "[{^11e}]", 1.0), "[1.000000e+00]");
    bad += differ(render(buf[:], "[{^11e}]", -1.0), "[-1.000000e+00]");
    bad += differ(render(buf[:], "[{^11e}]", 0.5), "[5.000000e-01]");
    bad += differ(render(buf[:], "[{^11e}]", 0.1), "[1.000000e-01]");
    bad += differ(render(buf[:], "[{^11e}]", -0.1), "[-1.000000e-01]");
    bad += differ(render(buf[:], "[{^11e}]", 123456.789), "[1.234568e+05]");
    bad += differ(render(buf[:], "[{^11e}]", -2.5), "[-2.500000e+00]");
    bad += differ(render(buf[:], "[{^11e}]", 1e+21), "[1.000000e+21]");
    bad += differ(render(buf[:], "[{^11e}]", 1e-07), "[1.000000e-07]");
    bad += differ(render(buf[:], "[{^11e}]", 5e-324), "[4.940656e-324]");
    bad += differ(render(buf[:], "[{^11e}]", 1.7976931348623157e+308), "[1.797693e+308]");
    bad += differ(render(buf[:], "[{^11e}]", 0.0), "[0.000000e+00]");
    bad += differ(render(buf[:], "[{^11e}]", -0.0), "[-0.000000e+00]");
    bad += differ(render(buf[:], "[{^11e}]", 100.0), "[1.000000e+02]");
    bad += differ(render(buf[:], "[{^11e}]", 0.000123), "[1.230000e-04]");
    bad += differ(render(buf[:], "[{^11e}]", 3.14159), "[3.141590e+00]");
    bad += differ(render(buf[:], "[{^11e}]", 9.995), "[9.995000e+00]");
    bad += differ(render(buf[:], "[{^11e}]", 0.375), "[3.750000e-01]");
    bad += differ(render(buf[:], "[{^11e}]", 1e+20), "[1.000000e+20]");
    bad += differ(render(buf[:], "[{^11e}]", 1e-05), "[1.000000e-05]");
    bad += differ(render(buf[:], "[{^11e}]", 99.5), "[9.950000e+01]");
    bad += differ(render(buf[:], "[{^11e}]", math.nan()), "[    NaN    ]");
    bad += differ(render(buf[:], "[{^11e}]", math.inf(1)), "[    inf    ]");
    bad += differ(render(buf[:], "[{^11e}]", math.inf(-1)), "[   -inf    ]");
    single += differ(render(buf[:], "{}", singles[0]), "1.1");
    single += differ(render(buf[:], "{.2f}", singles[0]), "1.10");
    single += differ(render(buf[:], "{}", singles[1]), "0.1");
    single += differ(render(buf[:], "{.2f}", singles[1]), "0.10");
    single += differ(render(buf[:], "{}", singles[2]), "2.5");
    single += differ(render(buf[:], "{.2f}", singles[2]), "2.50");
    single += differ(render(buf[:], "{}", singles[3]), "16777216");
    single += differ(render(buf[:], "{.2f}", singles[3]), "16777216.00");
    single += differ(render(buf[:], "{}", singles[4]), "3.4028235e+38");
    single += differ(render(buf[:], "{.2f}", singles[4]), "340282346638528859811704183484516925440.00");
    single += differ(render(buf[:], "{}", singles[5]), "1e-45");
    single += differ(render(buf[:], "{.2f}", singles[5]), "0.00");
    single += differ(render(buf[:], "{}", singles[6]), "0.3");
    single += differ(render(buf[:], "{.2f}", singles[6]), "0.30");
    single += differ(render(buf[:], "{}", singles[7]), "100");
    single += differ(render(buf[:], "{.2f}", singles[7]), "100.00");

    char[8] small;
    let _, problem = fmt.sprintf(small[:], "{}", 1234567.125);
    char[16] chars;
    let _, a = fmt.sprintf(chars[:], "{x}", 1.5);
    let _, b = fmt.sprintf(chars[:], "{#f}", 1.5);
    let _, c = fmt.sprintf(chars[:], "{.1001f}", 1.5);
    int rejects = (a != null ? 1 : 0) + (b != null ? 1 : 0) + (c != null ? 1 : 0);
    check(bad == 0, "floats match the reference for every spec and value");
    check(single == 0, "a float32 prints the shortest text of its own width");
    check(problem != null, "a float that does not fit the buffer is reported");
    check(rejects == 3, "a spec that is not for a float is reported");
}
`, expected: []string{"floats match the reference for every spec and value", "a float32 prints the shortest text of its own width", "a float that does not fit the buffer is reported", "a spec that is not for a float is reported"}},

	{dir: "float32_round", src: `func main() {
    float32 a = 0.1;
    float32 b = 0.2;
    float32 c = a + b;
    check(float(c) == 0.30000001192092896, "a float32 sum is rounded to single precision");

    float32 sum = 0.0;
    for i in 0..9 {
        sum += a;
    }
    check(float(sum) == 1.0000001192092896, "a float32 accumulator rounds at every step");

    float32 three = 3.0;
    float32 seven = 7.0;
    float32 q = a * three / seven;
    check(float(q) == 0.04285714402794838, "float32 multiplication and division round each result");

    int whole = 16777217;
    float32 widened = float32(whole);
    check(float(widened) == 16777216.0, "an int converts to the nearest float32");

    float32 huge = 1e30;
    float32 product = 1.5 * huge;
    check(float(product) == 1.500000098129063e+30, "a float32 product keeps single precision");

    float wide = 0.1;
    float32 narrowed = float32(wide);
    check(float(narrowed) == float(a), "narrowing a float rounds to the nearest float32");
}
`, expected: []string{"a float32 sum is rounded to single precision", "a float32 accumulator rounds at every step", "float32 multiplication and division round each result", "an int converts to the nearest float32", "a float32 product keeps single precision", "narrowing a float rounds to the nearest float32"}},

	{dir: "lambda_values", src: `int offset = 100;

struct Op {
    string name;
    (int, int) => int apply;
}

func twice((int) => int f, int v) int {
    return f(f(v));
}

type Step (int) => int;

func inc(int x) int => x + 1;
func dec(int x) int => x - 1;

func pick(bool up) (int) => int {
    return up ? inc : dec;
}

func both(int a, int b) (int, int) {
    return a + b, a - b;
}

func main() {
    let triple = (int x) int => x * 3;
    check(triple(5) == 15, "a function literal called through a variable");
    check(twice(inc, 1) == 3, "a named function passed as a value");
    check(twice((int x) int => x * 2, 3) == 12, "a function literal passed as an argument");
    check(pick(true)(10) == 11 && pick(false)(10) == 9, "a function returned from a function");

    Op add = Op{name: "add", apply: (int a, int b) int => a + b};
    check(add.apply(2, 3) == 5, "a function value in a struct field");

    let shifted = (int x) int => x + offset;
    check(shifted(1) == 101, "a function literal reads a global variable");

    (int, int) => (int, int) pair = both;
    let sum, diff = pair(7, 3);
    check(sum == 10 && diff == 4, "a function value with several results");

    Step[2] table = [inc, dec];
    check(table[0](5) == 6 && table[1](5) == 4, "an array of function values");

    (int) => int again = triple;
    again = inc;
    check(again(1) == 2, "a function variable takes another function");
}
`, expected: []string{"a function literal called through a variable", "a named function passed as a value", "a function literal passed as an argument", "a function returned from a function", "a function value in a struct field", "a function literal reads a global variable", "a function value with several results", "an array of function values", "a function variable takes another function"}},

	{dir: "lambda_globals", src: `type Step (int) => int;

func inc(int x) int => x + 1;
func dec(int x) int => x - 1;

Step chosen = inc;
Step made = (int x) int => x * 10;
Step missing = null;
Step[2] table = [inc, dec];
(int) => int plain = dec;

struct Hooks {
    (int) => int before;
}

Hooks hooks = Hooks{before: inc};

func apply(Step s, int v) int {
    return s(v);
}

func main() {
    check(chosen(1) == 2, "a global function variable starts as a function");
    check(made(2) == 20, "a global function variable starts as a function literal");
    check(table[0](5) == 6 && table[1](5) == 4, "a global array of function values");
    check(hooks.before(1) == 2, "a global struct with a function field");
    check(plain(1) == 0, "a global of a function type written in full");
    check(apply(inc, 1) == 2, "a function converts to a named function type");

    Step local = null;
    check(missing == null && local == null && chosen != null, "a function value is compared with null");
    chosen = null;
    check(chosen == null, "a function variable is set to null");
}
`, expected: []string{"a global function variable starts as a function", "a global function variable starts as a function literal", "a global array of function values", "a global struct with a function field", "a global of a function type written in full", "a function converts to a named function type", "a function value is compared with null", "a function variable is set to null"}},

	{dir: "pointer_decay", src: `func first(*int p) int {
    return p@;
}

func present(*void p) bool {
    return p != null;
}

func main() {
    int[4] arr = [7, 8, 9, 10];
    int[] tail = arr[1:];
    *int fromArray = arr;
    *int fromSlice = tail;
    *void untyped = tail;
    check(fromArray@ == 7, "an array converts to a pointer to its first element");
    check(fromSlice@ == 8, "a slice converts to a pointer to its first element");
    check((*int)(untyped)@ == 8, "a slice converts to a void pointer");
    check(first(arr) == 7 && first(tail) == 8, "an array and a slice are passed where a pointer is expected");
    check(present(arr) && present(tail), "an array and a slice are passed as a void pointer");

    char[8] text;
    char[] view = text[:];
    text[2] = 'z';
    *char at = view;
    check((at + 2)@ == 'z', "a slice of characters converts to a character pointer");
}
`, expected: []string{"an array converts to a pointer to its first element", "a slice converts to a pointer to its first element", "a slice converts to a void pointer", "an array and a slice are passed where a pointer is expected", "an array and a slice are passed as a void pointer", "a slice of characters converts to a character pointer"}},

	{dir: "mem_zeroed", src: `import "std/mem";

func dirty(*char p, int n) int {
    int count;
    for i in 0..n-1 {
        if p[i] != 0 {
            count++;
        }
    }
    return count;
}

func main() !int {
    *char a = mem.alloc(100)!;
    *char b = mem.alloc(100)!;
    *char c = mem.alloc(100)!;
    check(dirty(a, 100) == 0, "a new block is zeroed");
    mem.fill(a, 85, 100);
    mem.fill(b, 85, 100);
    mem.fill(c, 85, 100);
    check(a[0] == 85 && a[99] == 85 && c[50] == 85, "fill sets every byte");
    mem.free(a, b, c);

    *char x = mem.alloc(100)!;
    *char y = mem.alloc(100)!;
    *char z = mem.alloc(100)!;
    check(dirty(x, 100) + dirty(y, 100) + dirty(z, 100) == 0, "a reused block is zeroed");
    mem.free(x, y, z);

    *char grown = mem.alloc(16)!;
    mem.fill(grown, 7, 16);
    grown = mem.realloc(grown, 4000)!;
    check(grown[0] == 7 && grown[15] == 7, "realloc keeps the old contents");
    mem.free(grown);

    char[8] letters;
    mem.fill(letters, 'q', 8);
    check(letters[0] == 'q' && letters[7] == 'q', "fill takes an array");
    return 0;
}
`, expected: []string{"a new block is zeroed", "fill sets every byte", "a reused block is zeroed", "realloc keeps the old contents", "fill takes an array"}},

	{dir: "mem_limits", x64Only: true, src: `import "std/mem";
import "std/errors";

func main() {
    *void huge = mem.alloc(9223372036854775800) ?? null;
    *void over = mem.alloc((1 << 47) + 1) ?? null;
    check(huge == null && over == null, "a size that cannot exist is refused");
    *void hugeDirty = mem.allocDirty(9223372036854775800) ?? null;
    check(hugeDirty == null, "allocDirty refuses a size that cannot exist too");

    let _, problem = mem.alloc(9223372036854775800);
    check(problem == errors.Kind.OutOfMemory, "the refusal is OutOfMemory");

    *char a = mem.alloc(100) ?? null;
    *char b = mem.alloc(100) ?? null;
    *char c = mem.alloc(100) ?? null;
    mem.free(a, b, c);
    *char x = mem.alloc(100) ?? null;
    *char y = mem.alloc(100) ?? null;
    *char z = mem.alloc(100) ?? null;
    check(x == c && y == b && z == a, "freed blocks are reused without losing the list");

    *int big = mem.alloc(100000) ?? null;
    mem.fill(big, 1, 100000);
    mem.free(big);
    *int again = mem.alloc(100000) ?? null;
    check(again != null && again[0] == 0 && again[12000] == 0, "a large block is zeroed too");
}
`, expected: []string{"a size that cannot exist is refused", "allocDirty refuses a size that cannot exist too", "the refusal is OutOfMemory", "freed blocks are reused without losing the list", "a large block is zeroed too"}},

	{dir: "mem_move", src: `import "std/mem";

func fill(char[] t, string word) {
    for i, c in word {
        t[i] = c;
    }
}

func main() {
    char[8] a;
    char[8] b;
    char[8] c;
    fill(a, "abcdef");
    fill(b, "abcdef");
    fill(c, "abcdef");
    mem.move((*char)(a) + 2, a, 4);
    mem.move(b, (*char)(b) + 2, 4);
    mem.move((*char)(c) + 3, c, 0);
    check(string(a[0:6]) == "ababcd", "move copies forward over an overlap");
    check(string(b[0:6]) == "cdefef", "move copies backward over an overlap");
    check(string(c[0:6]) == "abcdef", "move of zero bytes changes nothing");

    char[8] apart;
    char[4] source;
    fill(source, "xyz");
    mem.move(apart, source, 3);
    check(string(apart[0:3]) == "xyz", "move works when the regions do not overlap");

    *char dirty = mem.allocDirty(64) ?? null;
    check(dirty != null, "allocDirty gives a block");
    mem.fill(dirty, 9, 64);
    check(dirty[0] == 9 && dirty[63] == 9, "a dirty block is usable");
    mem.free(dirty);
}
`, expected: []string{"move copies forward over an overlap", "move copies backward over an overlap", "move of zero bytes changes nothing", "move works when the regions do not overlap", "allocDirty gives a block", "a dirty block is usable"}},
	{dir: "any_error", src: `enum Kind {
    Bad = 1,
    Worse
}

func fail(int n) !int {
    if n < 0 {
        return Error(Kind.Bad, "negative");
    }
    if n == 0 {
        return Error(Kind.Worse);
    }
    return n;
}

func describe(...args) int {
    int seen = 0;
    for _, arg in args {
        switch let v = arg {
            case int:
                seen += v;
            case Error:
                if v == null {
                    check(true, "a null error is held as null");
                } else {
                    check(v.message == "negative" || v.message == "Kind.Worse", "an error keeps its message in an any");
                    check(v == Kind.Bad || v == Kind.Worse, "an error keeps its member in an any");
                }
                seen += 100;
            default:
                seen += 1000;
        }
    }
    return seen;
}

func main() {
    let _, e1 = fail(-1);
    let _, e2 = fail(0);
    let _, e3 = fail(1);
    check(describe(e1, e2, e3, 5) == 305, "errors and ints are told apart in an any");
    check(describe("x") == 1000, "a string is not an error");
}
`, expected: []string{"an error keeps its message in an any", "an error keeps its member in an any", "an error keeps its message in an any", "an error keeps its member in an any", "a null error is held as null", "errors and ints are told apart in an any", "a string is not an error"}},
	{dir: "method_error_param", src: `struct Box {
    int n;
}

func (*Box b) accept(Error e, int add) !int {
    if e != null {
        return Error(Kind.Bad, "refused");
    }
    return b.n + add;
}

func (*Box b) last() Error {
    return Error(Kind.Bad, "last");
}

enum Kind {
    Bad = 1
}

func main() {
    Box b = Box{n: 4};
    Error none;
    check((b.accept(none, 3) ?? -1) == 7, "a method takes an Error parameter");
    check((b.accept(b.last(), 3) ?? -1) == -1, "a method returns and receives an Error");
    check(b.last() == Kind.Bad, "a method returns an Error");
}
`, expected: []string{"a method takes an Error parameter", "a method returns and receives an Error", "a method returns an Error"}},
	{dir: "fmt_error", src: `import "std/fmt";
import "std/errors";

enum Fail {
    Bad = 1
}

func risky(int n) !int {
    if n < 0 {
        return Error(Fail.Bad, "negative input");
    }
    if n == 0 {
        return Error(errors.Kind.Invalid);
    }
    return n;
}

func text(char[] buf, string format, ...args) string {
    int n = fmt.sprintf(buf, format, args...)!!;
    return string(buf[0:n]);
}

func main() {
    let _, e1 = risky(-1);
    let _, e2 = risky(0);
    let _, none = risky(3);
    char[64] buf;
    check(text(buf, "{}", e1) == "negative input", "an error prints its message");
    check(text(buf, "{}", e2) == "Kind.Invalid", "a default message is the member");
    check(text(buf, "[{>16}]", e1) == "[  negative input]", "the message aligns like a string");
    check(text(buf, "{}", none) == "<null error>", "a null error prints a placeholder");
    check(text(buf, "{v}", none) == "<null error>", "a null error prints the same in verbose");
}
`, expected: []string{"an error prints its message", "a default message is the member", "the message aligns like a string", "a null error prints a placeholder", "a null error prints the same in verbose"}},
	{dir: "const_group_items", src: `const {
    uint BIG = 0x8000000000000000;
    int8 SMALL = 7;
    float RATIO = 1.5;
    LIMIT = 100;
    string NAME = "wisp";
}

const uint {
    LOW = 1;
    HIGH = 2;
}

func main() {
    uint big = BIG;
    int8 small = SMALL;
    float ratio = RATIO;
    int limit = LIMIT;
    uint low = LOW + HIGH;
    check(big == 9223372036854775808, "a typed item keeps its type in an untyped group");
    check(small == 7, "an int8 item works in a group");
    check(ratio == 1.5, "a float item works in a group");
    check(limit == 100, "an untyped item still works beside typed ones");
    check(NAME == "wisp", "a string item works in a group");
    check(low == 3, "a group type still applies to every item");
}
`, expected: []string{"a typed item keeps its type in an untyped group", "an int8 item works in a group", "a float item works in a group", "an untyped item still works beside typed ones", "a string item works in a group", "a group type still applies to every item"}},
	{dir: "mem_fixed", src: `import "std/mem";
import "std/errors";

func main() {
    char[128] storage;
    mem.fill(storage, 7, 128);
    mem.Fixed f = mem.Fixed{buffer: storage};

    *char a = f.alloc(10) ?? null;
    *char b = f.alloc(20) ?? null;
    check(a != null && b != null, "fixed gives blocks");
    check(int(b) - int(a) == 16, "a block is rounded up to 16 bytes");
    check(a[0] == 0 && a[9] == 0 && b[0] == 0 && b[19] == 0, "alloc zeroes the block");
    *char base = storage;
    check(int(a) - int(base) >= 0 && int(b) - int(base) < 128, "blocks live inside the buffer");

    *char c = f.allocDirty(48) ?? null;
    check(c != null, "allocDirty gives a block");
    check(int(c) - int(a) == 48, "blocks are contiguous");

    let full, e1 = f.alloc(40);
    check(full == null && e1 == errors.Kind.NoSpace, "a full buffer reports NoSpace");
    let neg, e2 = f.alloc(-1);
    check(neg == null && e2 == errors.Kind.Invalid, "a negative size is invalid");

    f.reset();
    *char d = f.alloc(10) ?? null;
    check(d == a, "reset reuses the buffer");
    check(d[0] == 0, "alloc zeroes memory that was used before");
}
`, expected: []string{"fixed gives blocks", "a block is rounded up to 16 bytes", "alloc zeroes the block", "blocks live inside the buffer", "allocDirty gives a block", "blocks are contiguous", "a full buffer reports NoSpace", "a negative size is invalid", "reset reuses the buffer", "alloc zeroes memory that was used before"}},
	{dir: "mem_arena", src: `import "std/mem";

func main() {
    mem.Arena a;
    *char p = a.alloc(10) ?? null;
    *char q = a.alloc(20) ?? null;
    check(p != null && q != null, "a zero arena works without init");
    check(int(q) - int(p) == 16, "arena blocks are rounded to 16 bytes");
    check(int(p) % 16 == 0 && int(q) % 16 == 0, "arena blocks are aligned");
    check(p[0] == 0 && p[9] == 0 && q[19] == 0, "arena alloc is zeroed");
    p[0] = 5;
    q[0] = 6;
    check(p[0] == 5 && q[0] == 6, "arena blocks do not overlap");

    int total = 0;
    bool ok = true;
    *char last = null;
    for i in 0..40 {
        *char b = a.alloc(5000) ?? null;
        if b == null {
            ok = false;
        } else {
            b[0] = char(i);
            b[4999] = char(i);
            total += 5008;
        }
    }
    check(ok, "arena grows over several chunks");
    check(total > 65536, "arena used more than one chunk");

    *char big = a.alloc(300000) ?? null;
    check(big != null, "arena serves a block bigger than a chunk");
    big[0] = 1;
    big[299999] = 2;
    check(big[0] == 1 && big[299999] == 2, "a big block is usable");
    *char after = a.alloc(10) ?? null;
    check(after != null && after[0] == 0, "arena keeps allocating after a big block");
    check(p[0] == 5 && q[0] == 6, "old blocks survive growth");

    let neg, e1 = a.alloc(-1);
    check(neg == null && e1 != null, "arena rejects a negative size");
}
`, expected: []string{"a zero arena works without init", "arena blocks are rounded to 16 bytes", "arena blocks are aligned", "arena alloc is zeroed", "arena blocks do not overlap", "arena grows over several chunks", "arena used more than one chunk", "arena serves a block bigger than a chunk", "a big block is usable", "arena keeps allocating after a big block", "old blocks survive growth", "arena rejects a negative size"}, x64Only: true},
	{dir: "mem_arena_reset", src: `import "std/mem";

func main() {
    mem.Arena a;
    bool ok = true;
    for i in 0..30 {
        *char b = a.alloc(5000) ?? null;
        if b == null {
            ok = false;
        } else {
            mem.fill(b, 9, 5000);
        }
    }
    *char big = a.alloc(200000) ?? null;
    check(ok && big != null, "arena filled before reset");
    mem.fill(big, 9, 200000);

    a.reset();
    *char x = a.alloc(100) ?? null;
    *char y = a.alloc(100) ?? null;
    check(x != null && y != null, "arena works after reset");
    check(int(y) - int(x) == 112, "reset starts a fresh chunk");
    bool clean = true;
    for i in 0..99 {
        if x[i] != 0 || y[i] != 0 {
            clean = false;
        }
    }
    check(clean, "alloc after reset is zeroed");

    for round in 0..4 {
        a.reset();
        bool good = true;
        for i in 0..20 {
            *char b = a.alloc(5000) ?? null;
            if b == null || b[0] != 0 {
                good = false;
            } else {
                b[0] = 1;
            }
        }
        check(good, "arena survives repeated resets");
    }

    a.reset();
    a.reset();
    *char again = a.alloc(10) ?? null;
    check(again != null, "reset twice in a row is harmless");

    mem.Arena empty;
    empty.reset();
    empty.free();
    *char fresh = empty.alloc(10) ?? null;
    check(fresh != null, "reset and free on an empty arena are harmless");

    a.free();
    *char z = a.alloc(64) ?? null;
    check(z != null && z[0] == 0 && z[63] == 0, "arena is usable after free");
    z[0] = 4;
    a.free();
    a.free();
    *char w = a.alloc(64) ?? null;
    check(w != null && w[0] == 0, "free twice in a row is harmless");
    a.free();
}
`, expected: []string{"arena filled before reset", "arena works after reset", "reset starts a fresh chunk", "alloc after reset is zeroed", "arena survives repeated resets", "arena survives repeated resets", "arena survives repeated resets", "arena survives repeated resets", "arena survives repeated resets", "reset twice in a row is harmless", "reset and free on an empty arena are harmless", "arena is usable after free", "free twice in a row is harmless"}, x64Only: true},
	{dir: "mem_arena_mark", src: `import "std/mem";

func main() {
    mem.Arena a;
    *char keep = a.alloc(100) ?? null;
    keep[0] = 42;
    keep[99] = 43;

    mem.Mark m = a.mark();
    *char t1 = a.alloc(200) ?? null;
    t1[0] = 1;
    a.rewind(m);
    *char t2 = a.alloc(200) ?? null;
    check(t1 == t2, "rewind reuses the space");
    check(t2[0] == 0, "alloc after rewind is zeroed");
    check(keep[0] == 42 && keep[99] == 43, "blocks before the mark survive");

    mem.Mark m2 = a.mark();
    bool ok = true;
    for i in 0..29 {
        *char b = a.alloc(5000) ?? null;
        if b == null {
            ok = false;
        } else {
            mem.fill(b, 9, 5000);
        }
    }
    check(ok, "arena grows after a mark");
    a.rewind(m2);
    *char t3 = a.alloc(100) ?? null;
    check(t3 == t2 + 208 && t3[0] == 0, "rewind across chunks returns to the marked spot");
    check(keep[0] == 42 && t2[0] == 0, "old blocks survive a rewind over several chunks");

    mem.Arena empty;
    mem.Mark e = empty.mark();
    *char x = empty.alloc(70000) ?? null;
    check(x != null, "an empty arena allocates after mark");
    empty.rewind(e);
    *char y = empty.alloc(10) ?? null;
    check(y != null && y[0] == 0, "rewind to an empty mark leaves a usable arena");

    mem.Mark nested1 = a.mark();
    *char n1 = a.alloc(64) ?? null;
    mem.Mark nested2 = a.mark();
    *char n2 = a.alloc(64) ?? null;
    a.rewind(nested2);
    *char n3 = a.alloc(64) ?? null;
    check(n3 == n2, "an inner rewind keeps the outer blocks");
    a.rewind(nested1);
    *char n4 = a.alloc(64) ?? null;
    check(n4 == n1, "an outer rewind drops them all");
    a.free();
    empty.free();
}
`, expected: []string{"rewind reuses the space", "alloc after rewind is zeroed", "blocks before the mark survive", "arena grows after a mark", "rewind across chunks returns to the marked spot", "old blocks survive a rewind over several chunks", "an empty arena allocates after mark", "rewind to an empty mark leaves a usable arena", "an inner rewind keeps the outer blocks", "an outer rewind drops them all"}, x64Only: true},
}
