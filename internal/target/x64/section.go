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

func (w *Writable) printt(str string) {
	fmt.Fprintf(&w.b, "\t%s", str)
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

func NewRodata() *rodataSection {
	section := &rodataSection{}
	section.println("section .rodata")
	return section
}

type dataSection struct {
	Writable
}

func NewData() *dataSection {
	section := &dataSection{}
	section.println("section .data")
	return section
}

type textSection struct {
	Writable
}

func NewText() *textSection {
	section := &textSection{}
	section.println("section .text")
	return section
}
