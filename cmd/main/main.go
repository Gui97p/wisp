package main

import (
	"fmt"
	"os"

	"github.com/Gui97p/wisp/internal/lexer"
)

func main() {
	args := os.Args
	if len(args) < 3 {
		fmt.Println("usage: wisp <module> <filename>.wsp")
		os.Exit(1)
	}

	buffer, err := os.ReadFile(args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	switch args[1] {
	case "tokens":
		l := lexer.NewLexer(buffer)
		token := l.NextToken()
		for token.Type != lexer.TOKEN_EOF {
			fmt.Printf("[%s] %s\n", token.Type.String(), token.Literal)
			token = l.NextToken()
		}
	default:
		fmt.Println("invalid module.\nAvaiable: tokens")
		os.Exit(1)
	}
}
