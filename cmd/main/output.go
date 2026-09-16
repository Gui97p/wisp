package main

import (
	"os"
	"path/filepath"
	"strings"
)

func resolveOutput(inputPath, output, outDir, ext string) (string, error) {
	if output == "" {
		base := strings.TrimSuffix(filepath.Base(inputPath), ".wsp")
		output = base + ext
	}
	if outDir == "" {
		outDir = "dist"
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(outDir, output), nil
}
