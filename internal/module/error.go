package module

import (
	"fmt"

	"github.com/Gui97p/wisp/internal/diag"
)

type SourceError struct {
	Path   string
	Buffer []byte
	Errors diag.List
}

func (e *SourceError) Error() string {
	return fmt.Sprintf("%s: %d error(s)", e.Path, len(e.Errors))
}
