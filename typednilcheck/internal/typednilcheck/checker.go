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
	Pos    token.Position
	Line   string
	Symbol string
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

func (v *fileVisitor) addErrorAtPosition(position token.Pos, symbol string) {
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
	v.errors = append(v.errors, PossibleTypedNilError{pos, line, symbol})
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
		retInterfName := make([]string, 0, len(t.Type.Results.List))
		retInterfIdx := make([]int, 0, len(t.Type.Results.List))
		for i, result := range t.Type.Results.List {
			tv := v.pkg.Types[result.Type]
			if !types.IsInterface(tv.Type) {
				continue
			}
			retInterfIdx = append(retInterfIdx, i)
			for _, name := range result.Names {
				retInterfName = append(retInterfName, name.Name)
			}
		}
		if len(retInterfIdx) == 0 {
			return nil
		}
		return &funcVisitor{
			fileVisitor:   v,
			retInterfIdex: retInterfIdx,
			retInterfName: retInterfName,
			ptrVar:        make(map[string]struct{}),
			nilVar:        make(map[string]struct{}),
		}
	}
	return v
}

type funcVisitor struct {
	fileVisitor   *fileVisitor
	retInterfIdex []int
	retInterfName []string
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
		assignedInIfStmt, nilInIfStmt := v.analysisIf(t, make(map[string]struct{}), make(map[string]struct{}), true)
		for assigned := range assignedInIfStmt {
			delete(v.nilVar, assigned)
		}
		for nilled := range nilInIfStmt {
			if _, ok := v.ptrVar[nilled]; ok {
				v.nilVar[nilled] = empty
			}
		}
		return nil
	case *ast.ReturnStmt:
		if len(v.retInterfIdex) == 0 {
			return nil
		}
		if len(t.Results) > 0 {
			for _, idx := range v.retInterfIdex {
				result := t.Results[idx]
				if ident, ok := result.(*ast.Ident); ok {
					if _, ok := v.nilVar[ident.Name]; ok {
						v.fileVisitor.addErrorAtPosition(ident.Pos(), ident.Name)
					}
				}
			}
			return nil
		}

		if len(v.retInterfName) > 0 {
			for _, retName := range v.retInterfName {
				if _, ok := v.nilVar[retName]; ok {
					v.fileVisitor.addErrorAtPosition(t.Pos(), retName)
				}
			}
		}

		return nil
	}
	return v
}

func (v *funcVisitor) analysisIf(ifStmt *ast.IfStmt, assigned map[string]struct{}, nilled map[string]struct{}, init bool) (map[string]struct{}, map[string]struct{}) {
	if ifStmt.Else == nil {
		assigned = make(map[string]struct{})
	}
	assignInBranch, nilInBranch := v.analysisAssignInBlock(ifStmt.Body)
	if init {
		for name, value := range assignInBranch {
			assigned[name] = value
		}
	} else {
		for name, _ := range assigned {
			if _, ok := assignInBranch[name]; !ok {
				delete(assigned, name)
			}
		}
	}
	for name, value := range nilInBranch {
		nilled[name] = value
	}

	if ifStmt.Else != nil {

		if elseIfBlock, ok := ifStmt.Else.(*ast.IfStmt); ok {
			assigned, nilled = v.analysisIf(elseIfBlock, assigned, nilled, false)
		}

		if elseBlock, ok := ifStmt.Else.(*ast.BlockStmt); ok {
			assignInBranch, nilInBranch := v.analysisAssignInBlock(elseBlock)
			for name, _ := range assigned {
				if _, ok := assignInBranch[name]; !ok {
					delete(assigned, name)
				}
			}
			for name, _ := range nilInBranch {
				nilled[name] = empty
			}
		}

	}
	return assigned, nilled
}

func (v *funcVisitor) analysisAssignInBlock(block *ast.BlockStmt) (map[string]struct{}, map[string]struct{}) {
	visitor := &BlockVisitor{
		funcVisitor:   v,
		assignInBlock: make(map[string]struct{}),
		nilInBlock:    make(map[string]struct{}),
	}
	ast.Walk(visitor, block)
	return visitor.assignInBlock, visitor.nilInBlock
}

func (v *funcVisitor) isNil(e ast.Expr) bool {
	return v.fileVisitor.pkg.Types[e].Type == types.Typ[types.UntypedNil]
}

type BlockVisitor struct {
	funcVisitor   *funcVisitor
	assignInBlock map[string]struct{}
	nilInBlock    map[string]struct{}
}

func (v *BlockVisitor) Visit(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.AssignStmt:
		for i, lexp := range t.Lhs {
			if lIdent, lok := lexp.(*ast.Ident); lok {
				isNil := v.funcVisitor.isNil(t.Rhs[i])
				if isNil {
					v.nilInBlock[lIdent.Name] = empty
					delete(v.assignInBlock, lIdent.Name)
				} else {
					v.assignInBlock[lIdent.Name] = empty
					delete(v.nilInBlock, lIdent.Name)
				}
			}
		}
		return nil
	case *ast.IfStmt:
		assignedInIfStmt, nilledInIfStmt := v.funcVisitor.analysisIf(t, make(map[string]struct{}), make(map[string]struct{}), true)
		for assigned := range assignedInIfStmt {
			v.assignInBlock[assigned] = empty
		}
		for nilled := range nilledInIfStmt {
			v.nilInBlock[nilled] = empty
		}
		return nil
	}
	return v
}
