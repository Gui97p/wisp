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

func resolveLuaPaths(root string, cfg *module.Config, tgt target.Target) (entry, objDir string) {
	entry = pick(binOut, rooted(root, pick(tgt.Output.Bin, cfg.Output.Bin)))
	if entry == "" {
		entry = filepath.Join(root, "bin", tgt.OS, tgt.Arch, filepath.Base(root)+".lua")
	}
	objDir = pick(objOut, rooted(root, pick(tgt.Output.Obj, cfg.Output.Obj)))
	if objDir == "" {
		objDir = filepath.Dir(entry)
	}
	return entry, objDir
}

func buildLua(root string, cfg *module.Config, tgt target.Target, forRun bool) (*buildResult, error) {
	entryPath, objDir := resolveLuaPaths(root, cfg, tgt)

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

	for _, c := range proj.compiled {
		isEntry := proj.mainMod != nil && c.mod == proj.mainMod.mod

		if !strings.HasPrefix(c.mod.Path, "std/") {
			c.info.RenderErrorBlocks(c.mod.BufferMap())
		}
		backend := lua.New(c.merged, c.info, isEntry)
		backend.SetFiles(c.declFiles)
		backend.SetSources(c.mod.BufferMap())
		backend.SetID(lua.ModuleID(c.mod.Path))
		backend.SetTarget(tgt.Name)
		source, err := backend.Compile()
		if err != nil {
			return nil, c.reportFailure(err)
		}

		outPath := entryPath
		if !isEntry {
			id := lua.ModuleID(c.mod.Path)
			outPath = filepath.Join(objDir, strings.ReplaceAll(id, ".", string(filepath.Separator))+".lua")
			if filepath.Clean(outPath) == filepath.Clean(entryPath) {
				return nil, fmt.Errorf("the module %q would be written over the entry script %s: give the entry another name with bin", c.mod.Path, relPath(entryPath))
			}
		} else if filepath.Clean(objDir) != filepath.Clean(filepath.Dir(entryPath)) {
			rel, err := filepath.Rel(filepath.Dir(entryPath), objDir)
			if err != nil {
				return nil, err
			}
			source = fmt.Sprintf("package.path = %q .. package.path\n", filepath.ToSlash(rel)+"/?.lua;") + source
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return nil, err
		}
		if err := lua.Build(source, outPath); err != nil {
			return nil, err
		}
	}

	detail("lua: %s, modules in %s", relPath(entryPath), relPath(objDir))

	return &buildResult{product: entryPath, luaDir: filepath.Dir(entryPath), luaEntry: entryPath}, nil
}
