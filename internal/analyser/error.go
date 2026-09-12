package analyser

import "fmt"

func (a *Analyser) ShowErrors() {
	for _, e := range a.errors {
		fmt.Println(e)
	}
}

func (a *Analyser) error(e string) {
	a.errors = append(a.errors, e)
}

func (a *Analyser) errorf(e string, args ...any) {
	a.error(fmt.Sprintf(e, args...))
}
