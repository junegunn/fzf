package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func loadPackages(t *testing.T) []*build.Package {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	var pkgs []*build.Package
	seen := make(map[string]bool)
	err = filepath.WalkDir(wd, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if name == "" || name[0] == '.' || name[0] == '_' || name == "vendor" || name == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && filepath.Ext(name) == ".go" && !strings.HasSuffix(name, "_test.go") {
			dir := filepath.Dir(path)
			if !seen[dir] {
				pkg, err := build.ImportDir(dir, build.ImportComment)
				if err != nil {
					return fmt.Errorf("%s: %s", dir, err)
				}
				if pkg.ImportPath == "" || pkg.ImportPath == "." {
					importPath, err := filepath.Rel(wd, dir)
					if err != nil {
						t.Fatal(err)
					}
					pkg.ImportPath = filepath.ToSlash(filepath.Join("github.com/junegunn/fzf", importPath))
				}

				pkgs = append(pkgs, pkg)
				seen[dir] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].ImportPath < pkgs[j].ImportPath
	})
	return pkgs
}

var sourceImporter = importer.ForCompiler(token.NewFileSet(), "source", nil)

func checkPackageForOsExit(t *testing.T, bpkg *build.Package, allowed map[string]int) (errOsExit bool) {
	var files []*ast.File
	fset := token.NewFileSet()
	for _, name := range bpkg.GoFiles {
		filename := filepath.Join(bpkg.Dir, name)
		af, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, af)
	}

	info := types.Info{
		Uses: make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{
		Importer: sourceImporter,
	}
	_, err := conf.Check(bpkg.Name, fset, files, &info)
	if err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for id, obj := range info.Uses {
		if obj.Pkg() != nil && obj.Pkg().Name() == "os" && obj.Name() == "Exit" {
			pos := fset.Position(id.Pos())

			name, err := filepath.Rel(wd, pos.Filename)
			if err != nil {
				t.Log(err)
				name = pos.Filename
			}
			name = filepath.ToSlash(name)

			// Check if the usage is allowed
			if allowed[name] > 0 {
				allowed[name]--
				continue
			}

			t.Errorf("os.Exit referenced at: %s:%d:%d", name, pos.Line, pos.Column)
			errOsExit = true
		}
	}
	return errOsExit
}

// Enforce that src/util.Exit() is used instead of os.Exit by prohibiting
// references to it anywhere else in the fzf code base.
func TestOSExitNotAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: short test")
	}
	allowed := map[string]int{
		"src/util/atexit.go": 1, // os.Exit allowed 1 time in "atexit.go"
	}
	var errOsExit bool
	for _, pkg := range loadPackages(t) {
		t.Run(pkg.ImportPath, func(t *testing.T) {
			if checkPackageForOsExit(t, pkg, allowed) {
				errOsExit = true
			}
		})
	}
	if t.Failed() && errOsExit {
		var names []string
		for name := range allowed {
			names = append(names, fmt.Sprintf("%q", name))
		}
		sort.Strings(names)

		const errMsg = `
Test failed because os.Exit was referenced outside of the following files:

    %s

Use github.com/junegunn/fzf/src/util.Exit() instead to exit the program.
This is enforced because calling os.Exit() prevents the functions
registered with util.AtExit() from running.`

		t.Errorf(errMsg, strings.Join(names, "\n    "))
	}
}
