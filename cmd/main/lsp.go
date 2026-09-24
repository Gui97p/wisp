package main

import (
	"github.com/Gui97p/wisp/internal/lsp"
	"github.com/spf13/cobra"
	"github.com/tliron/glsp/server"
)

var lspCmd = &cobra.Command{
	Use:          "lsp",
	Short:        "Run the Wisp language server",
	Args:         cobra.NoArgs,
	RunE:         runLsp,
	SilenceUsage: true,
}

func runLsp(cmd *cobra.Command, args []string) error {
	handler := lsp.NewHandler()
	srv := server.NewServer(handler, "wisp-lsp", false)
	return srv.RunStdio()
}
