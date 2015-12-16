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
	case *ast.BlockStmt:
		v.analysisAssignInBlock(nil, t)
	}
	return nil
}

func (v *funcVisitor) analysisAssignInBlock(parent *blockVisitor, block *ast.BlockStmt) (map[string]struct{}, map[string]struct{}) {
	return v.analysisAssignInStmts(parent, block.List)
}

func (v *funcVisitor) analysisAssignInStmts(parent *blockVisitor, stmts []ast.Stmt) (map[string]struct{}, map[string]struct{}) {
	visitor := &blockVisitor{
		parent:       parent,
		funcVisitor:  v,
		outAssign:    make(map[string]struct{}),
		outNil:       make(map[string]struct{}),
		innerDeclare: make(map[string]struct{}),
		innerAssign:  make(map[string]struct{}),
		innerNil:     make(map[string]struct{}),
	}
	for _, stmt := range stmts {
		ast.Walk(visitor, stmt)
	}
	return visitor.outAssign, visitor.outNil
}

func (v *funcVisitor) eqNil(e ast.Expr) bool {
	if v.fileVisitor.pkg.Types[e].Type == types.Typ[types.UntypedNil] {
		return true
	}
	switch t := e.(type) {
	case *ast.CallExpr:
		if parenExp, ok := t.Fun.(*ast.ParenExpr); ok && len(t.Args) == 1 {
			if _, ok := parenExp.X.(*ast.StarExpr); ok {
				if ident, ok := t.Args[0].(*ast.Ident); ok && ident.Name == "nil" {
					return true
				}
			}
		}
	}
	return false
}

type blockVisitor struct {
	funcVisitor  *funcVisitor
	parent       *blockVisitor
	outAssign    map[string]struct{}
	outNil       map[string]struct{}
	innerDeclare map[string]struct{}
	innerAssign  map[string]struct{}
	innerNil     map[string]struct{}
}

func (v *blockVisitor) checkAssign(name string) bool {
	return recurseAssign(v, name)
}

func recurseAssign(v *blockVisitor, name string) bool {
	if _, ok := v.innerAssign[name]; ok {
		return true
	}
	if v.parent != nil {
		parentNil := recurseAssign(v.parent, name)
		if parentNil {
			return true
		}
	}
	return false
}

func (v *blockVisitor) checkIsNil(name string) bool {
	return recurseNil(v, name)
}

func recurseNil(v *blockVisitor, name string) bool {
	if _, ok := v.innerNil[name]; ok {
		return true
	}
	if _, ok := v.innerAssign[name]; ok {
		return false
	}
	if _, ok := v.outNil[name]; ok {
		return true
	}
	if _, ok := v.outAssign[name]; ok {
		return false
	}
	if v.parent != nil {
		parentNil := recurseNil(v.parent, name)
		if parentNil {
			return true
		}
	}
	return false
}

func (v *blockVisitor) checkIsOutDeclared(name string) bool {
	if v.parent == nil {
		return false
	}
	return recurseDecl(v.parent, name)
}

func recurseDecl(v *blockVisitor, name string) bool {
	if _, ok := v.innerDeclare[name]; ok {
		return true
	}
	if v.parent != nil {
		parentNil := recurseDecl(v.parent, name)
		if parentNil {
			return true
		}
	}
	return false
}

func (v *blockVisitor) isNilPtr(e ast.Expr) bool {
	if v.funcVisitor.eqNil(e) {
		return true
	}
	switch t := e.(type) {
	case *ast.Ident:
		if v.checkIsNil(t.Name) {
			return true
		}
	}
	return false
}

func (v *blockVisitor) isAssignPrt(e ast.Expr) bool {
	if v.funcVisitor.eqNil(e) {
		return false
	}
	switch t := e.(type) {
	case *ast.UnaryExpr:
		if t.Op == token.AND {
			return true
		}
	case *ast.Ident:
		if v.checkAssign(t.Name) {
			return true
		}
	}

	return false
}

func (v *blockVisitor) Visit(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.DeclStmt:
		if gDel, ok := t.Decl.(*ast.GenDecl); ok {
			if gDel.Tok != token.VAR && gDel.Tok != token.CONST {
				return nil
			}
			for _, spec := range gDel.Specs {
				if varSpec, ok := spec.(*ast.ValueSpec); ok {
					ptrType := false
					if varSpec.Type != nil {
						_, ptrType = varSpec.Type.(*ast.StarExpr)
					}
					for i, name := range varSpec.Names {
						v.innerDeclare[name.Name] = empty
						var value *ast.Expr
						if len(varSpec.Values) > i {
							value = &(varSpec.Values[i])
						}
						if !ptrType && !(value != nil && v.isAssignPrt(*value)) {
							continue
						}
						if value == nil || v.isNilPtr(*value) {
							v.innerNil[name.Name] = empty
						}
					}
				}
			}
			return nil
		}
		return nil
	case *ast.AssignStmt:
		for i, lexp := range t.Lhs {
			if lIdent, lok := lexp.(*ast.Ident); lok {
				isDefine := t.Tok == token.DEFINE
				isNil := v.isNilPtr(t.Rhs[i])
				isAssign := v.isAssignPrt(t.Rhs[i])
				if isDefine && (isAssign || isNil) {
					v.innerDeclare[lIdent.Name] = empty
				}
				if _, ok := v.innerDeclare[lIdent.Name]; ok {
					if isNil {
						v.innerNil[lIdent.Name] = empty
						delete(v.innerAssign, lIdent.Name)
						continue
					}
					if isAssign {
						v.innerAssign[lIdent.Name] = empty
						delete(v.innerNil, lIdent.Name)
						continue
					}
					continue
				}
				if v.checkIsOutDeclared(lIdent.Name) {
					if isNil {
						v.outNil[lIdent.Name] = empty
						delete(v.outAssign, lIdent.Name)
						continue
					}
					if isAssign {
						v.outAssign[lIdent.Name] = empty
						delete(v.outNil, lIdent.Name)
						continue
					}
					continue
				}
			}
		}
		return nil
	case *ast.ReturnStmt:
		if len(v.funcVisitor.retInterfIdex) == 0 {
			return nil
		}
		if len(t.Results) > 0 {
			for _, idx := range v.funcVisitor.retInterfIdex {
				result := t.Results[idx]
				if ident, ok := result.(*ast.Ident); ok {
					if v.checkIsNil(ident.Name) {
						v.funcVisitor.fileVisitor.addErrorAtPosition(ident.Pos(), ident.Name)
					}
				}
			}
			return nil
		}

		if len(v.funcVisitor.retInterfName) > 0 {
			for _, retName := range v.funcVisitor.retInterfName {
				if _, ok := v.innerNil[retName]; ok {
					v.funcVisitor.fileVisitor.addErrorAtPosition(t.Pos(), retName)
				}
			}
		}

		return nil
	case *ast.SwitchStmt:
		assignedInSwitchStmt, nilledInSwitchStmt := v.analysisSwitch(t, make(map[string]struct{}), make(map[string]struct{}), true)
		for assigned := range assignedInSwitchStmt {
			if _, ok := v.innerDeclare[assigned]; ok {
				v.innerAssign[assigned] = empty
				delete(v.innerNil, assigned)
			} else if v.checkIsOutDeclared(assigned) {
				v.outAssign[assigned] = empty
				delete(v.outNil, assigned)
			}
		}
		for nilled := range nilledInSwitchStmt {
			if _, ok := v.innerDeclare[nilled]; ok {
				v.innerNil[nilled] = empty
				delete(v.innerAssign, nilled)
			} else if v.checkIsOutDeclared(nilled) {
				v.outNil[nilled] = empty
				delete(v.outAssign, nilled)
			}
			v.outNil[nilled] = empty
		}
		return nil
	case *ast.IfStmt:
		assignedInIfStmt, nilledInIfStmt := v.analysisIf(t, make(map[string]struct{}), make(map[string]struct{}), true)
		for assigned := range assignedInIfStmt {
			if _, ok := v.innerDeclare[assigned]; ok {
				v.innerAssign[assigned] = empty
				delete(v.innerNil, assigned)
			} else if v.checkIsOutDeclared(assigned) {
				v.outAssign[assigned] = empty
				delete(v.outNil, assigned)
			}
		}
		for nilled := range nilledInIfStmt {
			if _, ok := v.innerDeclare[nilled]; ok {
				v.innerNil[nilled] = empty
				delete(v.innerAssign, nilled)
			} else if v.checkIsOutDeclared(nilled) {
				v.outNil[nilled] = empty
				delete(v.outAssign, nilled)
			}
			v.outNil[nilled] = empty
		}
		return nil
	case *ast.ForStmt:
		assignInBlock, nilInBlock := v.funcVisitor.analysisAssignInBlock(v, t.Body)
		for assigned := range assignInBlock {
			if _, ok := v.innerDeclare[assigned]; ok {
				v.innerAssign[assigned] = empty
				delete(v.innerNil, assigned)
			} else if v.checkIsOutDeclared(assigned) {
				v.outAssign[assigned] = empty
				delete(v.outNil, assigned)
			}
		}
		for nilled := range nilInBlock {
			if _, ok := v.innerDeclare[nilled]; ok {
				v.innerNil[nilled] = empty
				delete(v.innerAssign, nilled)
			} else if v.checkIsOutDeclared(nilled) {
				v.outNil[nilled] = empty
				delete(v.outAssign, nilled)
			}
			v.outNil[nilled] = empty
		}
		return nil
	case *ast.RangeStmt:
		assignInBlock, nilInBlock := v.funcVisitor.analysisAssignInBlock(v, t.Body)
		for assigned := range assignInBlock {
			if _, ok := v.innerDeclare[assigned]; ok {
				v.innerAssign[assigned] = empty
				delete(v.innerNil, assigned)
			} else if v.checkIsOutDeclared(assigned) {
				v.outAssign[assigned] = empty
				delete(v.outNil, assigned)
			}
		}
		for nilled := range nilInBlock {
			if _, ok := v.innerDeclare[nilled]; ok {
				v.innerNil[nilled] = empty
				delete(v.innerAssign, nilled)
			} else if v.checkIsOutDeclared(nilled) {
				v.outNil[nilled] = empty
				delete(v.outAssign, nilled)
			}
			v.outNil[nilled] = empty
		}
		return nil
	}
	return v
}

func (v *blockVisitor) analysisSwitch(switchStmt *ast.SwitchStmt, assigned map[string]struct{}, nilled map[string]struct{}, init bool) (map[string]struct{}, map[string]struct{}) {

	innnVisitor := v
	if switchStmt.Init != nil {
		switch t := switchStmt.Init.(type) {
		case *ast.AssignStmt:
			innnVisitor = &blockVisitor{
				parent:       v,
				funcVisitor:  v.funcVisitor,
				outAssign:    make(map[string]struct{}),
				outNil:       make(map[string]struct{}),
				innerDeclare: make(map[string]struct{}),
				innerAssign:  make(map[string]struct{}),
				innerNil:     make(map[string]struct{}),
			}
			ast.Walk(innnVisitor, t)
		}
	}

	hasDefault := false
	for idx, blockStmt := range switchStmt.Body.List {
		if causeStmt, ok := blockStmt.(*ast.CaseClause); ok {
			hasDefault = (causeStmt.List == nil)
			assignInBranch, nilInBranch := innnVisitor.funcVisitor.analysisAssignInStmts(innnVisitor, causeStmt.Body)
			if idx == 0 {
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
		}
	}

	if !hasDefault {
		assigned = make(map[string]struct{})
	}

	if len(innnVisitor.outNil) != 0 {
		for initNil  := range innnVisitor.outNil {
			if _, ok := assigned[initNil]; !ok {
				nilled[initNil] = empty
			}
		}
		for assignNil := range innnVisitor.outAssign {
			if _, ok := nilled[assignNil]; !ok {
				assigned[assignNil] = empty
			}
		}
	}

	return assigned, nilled
}

func (v *blockVisitor) analysisIf(ifStmt *ast.IfStmt, assigned map[string]struct{}, nilled map[string]struct{}, init bool) (map[string]struct{}, map[string]struct{}) {

	innnVisitor := v
	if ifStmt.Init != nil {
		switch t := ifStmt.Init.(type) {
		case *ast.AssignStmt:
			innnVisitor = &blockVisitor{
				parent:       v,
				funcVisitor:  v.funcVisitor,
				outAssign:    make(map[string]struct{}),
				outNil:       make(map[string]struct{}),
				innerDeclare: make(map[string]struct{}),
				innerAssign:  make(map[string]struct{}),
				innerNil:     make(map[string]struct{}),
			}
			ast.Walk(innnVisitor, t)
		}
	}

	assignInBranch, nilInBranch := innnVisitor.funcVisitor.analysisAssignInBlock(innnVisitor, ifStmt.Body)
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
			assigned, nilled = innnVisitor.analysisIf(elseIfBlock, assigned, nilled, false)
		}

		if elseBlock, ok := ifStmt.Else.(*ast.BlockStmt); ok {
			assignInBranch, nilInBranch := innnVisitor.funcVisitor.analysisAssignInBlock(innnVisitor, elseBlock)
			for name, _ := range assigned {
				if _, ok := assignInBranch[name]; !ok {
					delete(assigned, name)
				}
			}
			for name, _ := range nilInBranch {
				nilled[name] = empty
			}
		}

	} else {
		assigned = make(map[string]struct{})
	}

	if len(innnVisitor.outNil) != 0 {
		for initNil  := range innnVisitor.outNil {
			if _, ok := assigned[initNil]; !ok {
				nilled[initNil] = empty
			}
		}
		for assignNil := range innnVisitor.outAssign {
			if _, ok := nilled[assignNil]; !ok {
				assigned[assignNil] = empty
			}
		}
	}

	return assigned, nilled
}
