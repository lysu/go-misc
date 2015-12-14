package typednilcheck

import (
	"bufio"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/types"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var empty = struct{}{}

var ErrNoGoFiles = errors.New("package contains no go source files")

type Checker struct {
	WithoutTests bool
}

type PossibleTypedNilError struct {
	Pos  token.Position
	Line string
}

type PossibleTypedNilErrors struct {
	Errors []PossibleTypedNilError
}

func (e PossibleTypedNilErrors) Error() string {
	return fmt.Sprintf("%d unchecked errors", len(e.Errors))
}

func (e PossibleTypedNilErrors) Len() int { return len(e.Errors) }

func (e PossibleTypedNilErrors) Swap(i, j int) { e.Errors[i], e.Errors[j] = e.Errors[j], e.Errors[i] }

type byName struct{ PossibleTypedNilErrors }

func (e byName) Less(i, j int) bool {
	ei, ej := e.Errors[i], e.Errors[j]

	pi, pj := ei.Pos, ej.Pos

	if pi.Filename != pj.Filename {
		return pi.Filename < pj.Filename
	}
	if pi.Line != pj.Line {
		return pi.Line < pj.Line
	}
	if pi.Column != pj.Column {
		return pi.Column < pj.Column
	}

	return ei.Line < ej.Line
}

func (c *Checker) logf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
}

func (c *Checker) CheckPackages(pathes ...string) error {
	ctx := build.Default
	loadcfg := loader.Config{
		Build: &ctx,
	}
	rest, err := loadcfg.FromArgs(pathes, !c.WithoutTests)
	if err != nil {
		return fmt.Errorf("could not parse arguments: %s", err)
	}
	if len(rest) > 0 {
		return fmt.Errorf("unhandled extra arguments: %v", rest)
	}

	program, err := loadcfg.Load()
	if err != nil {
		return fmt.Errorf("could not type check: %s", err)
	}

	var errsMutex sync.Mutex
	var errs []PossibleTypedNilError

	var wg sync.WaitGroup

	for _, pkgInfo := range program.InitialPackages() {
		if pkgInfo.Pkg.Path() == "unsafe" { // not a real package
			continue
		}

		wg.Add(1)

		go func(pkgInfo *loader.PackageInfo) {
			defer wg.Done()
			c.logf("Checking %s", pkgInfo.Pkg.Path())

			v := &fileVisitor{
				prog:   program,
				pkg:    pkgInfo,
				lines:  make(map[string][]string),
				errors: []PossibleTypedNilError{},
			}

			for _, astFile := range v.pkg.Files {
				ast.Walk(v, astFile)
			}
			if len(v.errors) > 0 {
				errsMutex.Lock()
				defer errsMutex.Unlock()

				errs = append(errs, v.errors...)
			}
			c.logf("Check %s done.", pkgInfo.Pkg.Path())
		}(pkgInfo)
	}

	wg.Wait()

	if len(errs) > 0 {
		u := PossibleTypedNilErrors{errs}
		sort.Sort(byName{u})
		return u
	}

	return nil

}

type fileVisitor struct {
	prog    *loader.Program
	pkg     *loader.PackageInfo
	ignore  map[string]*regexp.Regexp
	blank   bool
	asserts bool
	lines   map[string][]string

	errors []PossibleTypedNilError
}

func (v *fileVisitor) addErrorAtPosition(position token.Pos) {
	pos := v.prog.Fset.Position(position)
	lines, ok := v.lines[pos.Filename]
	if !ok {
		lines = readfile(pos.Filename)
		v.lines[pos.Filename] = lines
	}

	line := "??"
	if pos.Line-1 < len(lines) {
		line = strings.TrimSpace(lines[pos.Line-1])
	}
	v.errors = append(v.errors, PossibleTypedNilError{pos, line})
}

func readfile(filename string) []string {
	var f, err = os.Open(filename)
	if err != nil {
		return nil
	}

	var lines []string
	var scanner = bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func (v *fileVisitor) Visit(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.FuncDecl:
		if t.Type.Results == nil {
			return nil
		}
		retInterfIdx := make([]int, 0, len(t.Type.Results.List))
		for i, result := range t.Type.Results.List {
			tv := v.pkg.Types[result.Type]
			if types.IsInterface(tv.Type) {
				retInterfIdx = append(retInterfIdx, i)
			}
		}
		if len(retInterfIdx) == 0 {
			return nil
		}
		return &funcVisitor{
			fileVisitor:   v,
			retInterfIdex: retInterfIdx,
			ptrVar:        make(map[string]struct{}),
			nilVar:        make(map[string]struct{}),
		}
	}
	return v
}

type funcVisitor struct {
	fileVisitor   *fileVisitor
	retInterfIdex []int
	ptrVar        map[string]struct{}
	nilVar        map[string]struct{}
}

func (v *funcVisitor) Visit(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.DeclStmt:
		if gDel, ok := t.Decl.(*ast.GenDecl); ok {
			if gDel.Tok != token.VAR {
				return nil
			}
			for _, spec := range gDel.Specs {
				if varSpec, ok := spec.(*ast.ValueSpec); ok {
					if _, ok := varSpec.Type.(*ast.StarExpr); !ok {
						continue
					}
					for i, name := range varSpec.Names {
						v.ptrVar[name.Name] = empty
						var value *ast.Expr
						if len(varSpec.Values) > i {
							value = &(varSpec.Values[i])
						}
						if value == nil || v.isNil(*value) {
							v.nilVar[name.Name] = empty
						}
					}
				}
			}
			return nil
		}
	case *ast.AssignStmt:
		for i, lexp := range t.Lhs {
			if lIdent, lok := lexp.(*ast.Ident); lok {
				isNil := v.isNil(t.Rhs[i])
				if !isNil {
					delete(v.nilVar, lIdent.Name)
				} else {
					if _, ok := v.ptrVar[lIdent.Name]; ok {
						v.nilVar[lIdent.Name] = empty
					}
				}
			}
		}
		return nil
	case *ast.IfStmt:
		return &ifVisitor{
			funcVisitor: v,
			assignedVar: make(map[string]struct{}),
			nilVar:      make(map[string]struct{}),
		}
	case *ast.ReturnStmt:
		if len(v.retInterfIdex) == 0 {
			return nil
		}
		for _, idx := range v.retInterfIdex {
			result := t.Results[idx]
			if ident, ok := result.(*ast.Ident); ok {
				if _, ok := v.nilVar[ident.Name]; ok {
					v.fileVisitor.addErrorAtPosition(ident.Pos())
				}
			}
		}
		return nil
	}
	return v
}

func (v *funcVisitor) isNil(e ast.Expr) bool {
	return v.fileVisitor.pkg.Types[e].Type == types.Typ[types.UntypedNil]
}

type ifVisitor struct {
	funcVisitor *funcVisitor
	assignedVar map[string]struct{}
	nilVar      map[string]struct{}
}

func (v *ifVisitor) Visit(node ast.Node) ast.Visitor {
	return v
}
