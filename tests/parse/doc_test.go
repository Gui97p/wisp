package parse

import (
	"testing"

	"github.com/Gui97p/wisp/internal/ast"
)

func TestDocumentationComments(t *testing.T) {
	program, errors := parse(t, `/// Adds two numbers.
/// Both are copied.
func add(int a, int b) int {
	return a + b;
}

// A plain comment is not documentation.
func plain() {}

/// The doc comment survives a plain comment between it and the declaration.
// TODO: rename
func kept() {}

/// Not attached: a blank line separates it from the declaration.

func detached() {}

/// A point.
struct Point {
	/// The horizontal position.
	int x;
	// Not a doc.
	int y;
}

/// Colors.
enum Color {
	/// The warm one.
	Red,
	// Not a doc.
	Green
}

/// Writes things.
interface Writer {
	/// Sends n bytes.
	func write(int n) int;
	// Not a doc.
	func flush();
}
`)
	if len(errors) > 0 {
		t.Fatalf("errors: %v", errors)
	}

	docs := map[string]string{}
	for _, d := range program.Declarations {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			docs[decl.Name] = decl.Doc
		case *ast.StructDecl:
			docs[decl.Name] = decl.Doc
			for _, m := range decl.Members {
				docs["Point."+m.Name] = m.Doc
			}
		case *ast.TypeDecl:
			docs[decl.Name] = decl.Doc
			for _, m := range decl.Members {
				docs["Color."+m.Name] = m.Doc
			}
		case *ast.InterfaceDecl:
			docs[decl.Name] = decl.Doc
			for _, m := range decl.Methods {
				docs["Writer."+m.Name] = m.Doc
			}
		}
	}

	want := map[string]string{
		"add":          "Adds two numbers.\nBoth are copied.",
		"plain":        "",
		"kept":         "The doc comment survives a plain comment between it and the declaration.",
		"detached":     "",
		"Point":        "A point.",
		"Point.x":      "The horizontal position.",
		"Point.y":      "",
		"Color":        "Colors.",
		"Color.Red":    "The warm one.",
		"Color.Green":  "",
		"Writer":       "Writes things.",
		"Writer.write": "Sends n bytes.",
		"Writer.flush": "",
	}
	for name, doc := range want {
		if docs[name] != doc {
			t.Errorf("%s: doc %q, want %q", name, docs[name], doc)
		}
	}
}
