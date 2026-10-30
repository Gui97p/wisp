package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target"
	"github.com/Gui97p/wisp/internal/target/lua"
)

func resolveLuaOutDir(root string, cfg *module.Config, tgt target.Target) string {
	dir := pick(luaOut, tgt.Output.Lua, cfg.Output.Lua)
	if dir == "" {
		return filepath.Join(root, "dist", tgt.Name)
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(root, dir)
}

func buildLua(root string, cfg *module.Config, tgt target.Target, forRun bool) (*buildResult, error) {
	outDir := resolveLuaOutDir(root, cfg, tgt)

	analyseStart := time.Now()
	proj, err := loadProject(root, cfg)
	if err != nil {
		return nil, err
	}
	detail("Analysed %s in %s", count(len(proj.compiled), "module"), elapsed(analyseStart))

	if forRun {
		if err := proj.requireMain(); err != nil {
			return nil, fmt.Errorf("cannot run, %w", err)
		}
	}
	if err := proj.rejectManyMains(); err != nil {
		return nil, err
	}

	var entryOutputPath string

	for _, c := range proj.compiled {
		isEntry := proj.mainMod != nil && c.mod == proj.mainMod.mod

		backend := lua.New(c.merged, c.info, isEntry)
		backend.SetFiles(c.declFiles)
		backend.SetID(lua.ModuleID(c.mod.Path))
		backend.SetTarget(tgt.Name)
		source, err := backend.Compile()
		if err != nil {
			return nil, c.reportFailure(err)
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

	detail("lua: %s", relPath(outDir))

	return &buildResult{product: outDir, luaDir: outDir, luaEntry: entryOutputPath}, nil
}
