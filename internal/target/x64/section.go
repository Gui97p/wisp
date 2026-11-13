package x64

import (
	"fmt"
	"strings"
)

type Writable struct {
	b strings.Builder
}

func (w *Writable) print(str string) {
	w.b.WriteString(str)
}

func (w *Writable) printf(str string, args ...any) {
	fmt.Fprintf(&w.b, str, args...)
}

func (w *Writable) println(str string) {
	fmt.Fprintf(&w.b, "%s\n", str)
}

func (w *Writable) printft(str string, args ...any) {
	fmt.Fprintf(&w.b, "\t"+str, args...)
}

func (w *Writable) printlnt(str string) {
	fmt.Fprintf(&w.b, "\t%s\n", str)
}

func (w *Writable) newLine() {
	w.b.WriteRune('\n')
}

type prelude struct {
	extern map[string]bool
	Writable
}

func (p *prelude) define(ident string) {
	if _, ok := p.extern[ident]; ok {
		return
	}
	p.printf("extern %s\n", ident)
	p.extern[ident] = true
}

func NewPrelude() *prelude {
	p := &prelude{extern: map[string]bool{}}
	return p
}

type postlude struct {
	defined map[string]bool
	Writable
}

func (p *postlude) define(ident string) bool {
	if _, ok := p.defined[ident]; ok {
		return false
	}
	p.defined[ident] = true
	return true
}

func NewPostlude() *postlude {
	p := &postlude{defined: map[string]bool{}}
	return p
}

type rodataSection struct {
	Writable
}

func rodataOf(name string) (string, error) {
	switch name {
	case "elf64":
		return "align=16", nil
	case "win64":
		return "rdata align=16", nil
	}
	return "", fmt.Errorf("x86-64: unknown format")
}

func NewRodata(format string) (*rodataSection, error) {
	section := &rodataSection{}
	r, err := rodataOf(format)
	if err != nil {
		return nil, err
	}
	if r != "" {
		r = " " + r
	}
	section.printf("section .rodata%s\n", r)
	return section, nil
}

type dataSection struct {
	Writable
}

func dataOf(name string) (string, error) {
	switch name {
	case "elf64":
		return "", nil
	case "win64":
		return "data align=8", nil
	}
	return "", fmt.Errorf("x86-64: unknown format")
}

func NewData(format string) (*dataSection, error) {
	section := &dataSection{}
	d, err := dataOf(format)
	if err != nil {
		return nil, err
	}
	if d != "" {
		d = " " + d
	}
	section.printf("section .data%s\n", d)
	return section, nil
}

type textSection struct {
	Writable
}

func textOf(name string) (string, error) {
	switch name {
	case "elf64":
		return "", nil
	case "win64":
		return "code align=16", nil
	}
	return "", fmt.Errorf("x86-64: unknown format")
}

func NewText(format string) (*textSection, error) {
	section := &textSection{}
	t, err := textOf(format)
	if err != nil {
		return nil, err
	}
	if t != "" {
		t = " " + t
	}
	section.printf("section .text%s\n", t)
	return section, nil
}

func newDesc(format, name, leader string, exec bool, align int) (string, error) {
	al := fmt.Sprintf("align=%d", align)
	switch format {
	case "elf64":
		e := "noexec"
		if exec {
			e = "exec"
		}
		return fmt.Sprintf(".%s progbits alloc %s nowrite %s", name, e, al), nil
	case "win64":
		e := "rdata"
		if exec {
			e = "code"
		}
		return fmt.Sprintf("$%s %s %s comdat=2:%s", name, e, al, leader), nil
	}
	return "", fmt.Errorf("x86-64: unknown format")
}
