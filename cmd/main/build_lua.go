package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target"
	"github.com/Gui97p/wisp/internal/target/lua"
)

func resolveLuaPaths(root string, cfg *module.Config, tgt target.Target) (entry, objDir string) {
	name := binaryNameOf(root, cfg)
	binDir := pickPath(root, tgt, name, binOut, cfg.ProgramOutput.Bin, cfg.Output.Bin)
	if binDir == "" {
		binDir = filepath.Join(root, "bin", profileDir(tgt))
	}
	entry = filepath.Join(binDir, name+".lua")
	objDir = pickPath(root, tgt, name, objOut, cfg.ProgramOutput.Obj, cfg.Output.Obj)
	if objDir == "" {
		objDir = binDir
	}
	return entry, objDir
}

func buildLua(root string, cfg *module.Config, tgt target.Target, forRun bool, job buildJob) (*buildResult, error) {
	entryPath, objDir := resolveLuaPaths(root, cfg, tgt)

	analyseStart := time.Now()
	proj, err := loadProject(root, cfg)
	if err != nil {
		return nil, err
	}
	detail("Analysed %s in %s", count(len(proj.compiled), "module"), elapsed(analyseStart))

	if job.library {
		roots, err := libraryRoots(root, job.name, cfg.Libraries[job.name].Modules)
		if err != nil {
			return nil, err
		}
		if err := proj.restrict(roots); err != nil {
			return nil, fmt.Errorf("the library %s: %w", job.name, err)
		}
	} else {
		if forRun {
			if err := proj.requireMain(); err != nil {
				return nil, fmt.Errorf("cannot run, %w", err)
			}
		}
		if err := proj.rejectManyMains(); err != nil {
			return nil, err
		}
		if proj.mainMod != nil {
			if err := proj.restrict([]string{proj.mainMod.mod.Path}); err != nil {
				return nil, err
			}
		}
	}

	for _, c := range proj.compiled {
		isEntry := proj.mainMod != nil && c.mod == proj.mainMod.mod

		if !strings.HasPrefix(c.mod.Path, "std/") {
			renderErrorBlocks(c)
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
				return nil, fmt.Errorf("the module %q would be written over the entry script %s: give the program another name", c.mod.Path, relPath(entryPath))
			}
		} else if filepath.Clean(objDir) != filepath.Clean(filepath.Dir(entryPath)) {
			rel, err := filepath.Rel(filepath.Dir(entryPath), objDir)
			if err != nil {
				return nil, err
			}
			source = fmt.Sprintf("package.path = %q .. package.path\n", filepath.ToSlash(rel)+"/?.lua;") + source
		}

		if err := makeOutputDir(filepath.Dir(outPath)); err != nil {
			return nil, err
		}
		if err := lua.Build(source, outPath); err != nil {
			return nil, err
		}
	}

	detail("lua: %s, modules in %s", relPath(entryPath), relPath(objDir))

	if job.library {
		return &buildResult{product: objDir}, nil
	}
	return &buildResult{product: entryPath, luaDir: filepath.Dir(entryPath), luaEntry: entryPath}, nil
}
