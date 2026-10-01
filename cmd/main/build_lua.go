package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target/lua"
)

func resolveLuaOutDir(root string, cfg *module.Config) string {
	dir := luaOut
	if dir == "" {
		dir = cfg.Output.Lua
	}
	if dir == "" {
		return filepath.Join(root, "dist")
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(root, dir)
}

func buildLua(root string, cfg *module.Config, forRun bool) (*buildResult, error) {
	outDir := resolveLuaOutDir(root, cfg)

	proj, err := loadProject(root, cfg)
	if err != nil {
		return nil, err
	}

	if forRun {
		if err := proj.requireMain(); err != nil {
			return nil, fmt.Errorf("cannot run: %w", err)
		}
	}
	if err := proj.rejectManyMains(); err != nil {
		return nil, err
	}

	var entryOutputPath string

	for _, c := range proj.compiled {
		isEntry := proj.mainMod != nil && c.mod == proj.mainMod.mod

		source, err := lua.New(c.merged, c.info, isEntry).Compile()
		if err != nil {
			return nil, err
		}

		id := lua.ModuleID(c.mod.Path)
		outPath := filepath.Join(outDir, strings.ReplaceAll(id, ".", string(filepath.Separator))+".lua")

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return nil, err
		}
		if err := lua.Build(source, outPath); err != nil {
			return nil, err
		}

		if isEntry {
			entryOutputPath = outPath
		}
	}

	fmt.Printf("[lua] built %s\n", outDir)

	return &buildResult{luaDir: outDir, luaEntry: entryOutputPath}, nil
}
