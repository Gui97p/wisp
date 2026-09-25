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
