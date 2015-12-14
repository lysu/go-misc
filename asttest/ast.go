package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
)

var empty = struct{}{}

type testVisitor struct {
	info types.Info
}

func (v *testVisitor) ifMethodReturnInterface(node ast.Node, funcDecl func(t *ast.FuncDecl, interfaceRet []int)) {
	switch t := node.(type) {
	case *ast.FuncDecl:
		if t.Type.Results == nil {
			return
		}
		results := t.Type.Results.List
		interfaceIndexs := make([]int, 0, len(results))
		for i, result := range results {
			tv := v.info.Types[result.Type]
			if types.IsInterface(tv.Type) {
				interfaceIndexs = append(interfaceIndexs, i)
			}
		}
		if len(interfaceIndexs) == 0 {
			return
		}
		funcDecl(t, interfaceIndexs)
	}
}

func (v *testVisitor) ifDeclVar(stmt ast.Stmt, genDecl func(genDecl *ast.GenDecl)) {
	if decl, ok := stmt.(*ast.DeclStmt); ok {
		if gDel, ok := decl.Decl.(*ast.GenDecl); ok {
			if gDel.Tok != token.VAR {
				return
			}
			genDecl(gDel)
		}
	}
}

func (v *testVisitor) ifAssign(stmt ast.Stmt, assgin func(assign *ast.AssignStmt)) {
	if asStmt, ok := stmt.(*ast.AssignStmt); ok {
		assgin(asStmt)
	}
}

func (v *testVisitor) ifIf(stmt ast.Stmt, fullIf func(ifStmt *ast.IfStmt)) {
	if ifStmt, ok := stmt.(*ast.IfStmt); ok {
		fullIf(ifStmt)
	}
}

func (v *testVisitor) Visit(node ast.Node) (w ast.Visitor) {
	v.ifMethodReturnInterface(node, func(t *ast.FuncDecl, interfaceRet []int) {

		mayNilVars := make(map[string]struct{})
		starTypeVars := make(map[string]struct{})
		for _, stmt := range t.Body.List {
			v.ifDeclVar(stmt, func(genDecl *ast.GenDecl) {
				for _, spec := range genDecl.Specs {
					if varSpec, ok := spec.(*ast.ValueSpec); ok {
						if _, ok := varSpec.Type.(*ast.StarExpr); !ok {
							continue
						}
						for i, name := range varSpec.Names {
							starTypeVars[name.Name] = empty
							var value *ast.Expr
							if len(varSpec.Values) > i {
								value = &(varSpec.Values[i])
							}
							if value == nil || v.isNil(*value) {
								mayNilVars[name.Name] = empty
							}
						}
					}
				}
			})

			v.ifAssign(stmt, func(assign *ast.AssignStmt) {
				for i, lexp := range assign.Lhs {
					if lIdent, lok := lexp.(*ast.Ident); lok {
						isNil := v.isNil(assign.Rhs[i])
						if !isNil {
							delete(mayNilVars, lIdent.Name)
						} else {
							if _, ok := starTypeVars[lIdent.Name]; ok {
								mayNilVars[lIdent.Name] = empty
							}
						}
					}
				}
			})

			v.ifIf(stmt, func(ifStmt *ast.IfStmt) {
				assignedInIfStmt,nilInIfStmt := v.analysisIf(ifStmt, make(map[string]struct{}), make(map[string]struct{}), true)
				for assigned := range assignedInIfStmt {
					delete(mayNilVars, assigned)
				}
				for nilled := range nilInIfStmt {
					if _, ok := starTypeVars[nilled]; ok {
						mayNilVars[nilled] = empty
					}
				}
			})

			if retStmt, ok := stmt.(*ast.ReturnStmt); ok {
				if len(interfaceRet) == 0 {
					return
				}
				for _, idx := range interfaceRet {
					result := retStmt.Results[idx]
					if ident, ok := result.(*ast.Ident); ok {
						if _, ok := mayNilVars[ident.Name]; ok {
							fmt.Printf("maybe return [%s] nil\n", ident.Name)
						}
					}
				}
			}
		}
	})
	return v
}

func (v *testVisitor) analysisAssignInBlock(stmts []ast.Stmt) (map[string]struct{}, map[string]struct{}) {
	assignInBlock := make(map[string]struct{})
	nilInBlock := make(map[string]struct{})
	for _, bStmt := range stmts {
		v.ifAssign(bStmt, func(assign *ast.AssignStmt) {
			for i, lexp := range assign.Lhs {
				if lIdent, lok := lexp.(*ast.Ident); lok {
					isNil := v.isNil(assign.Rhs[i])
					if isNil {
						nilInBlock[lIdent.Name] = empty
						delete(assignInBlock, lIdent.Name)
					} else {
						assignInBlock[lIdent.Name] = empty
						delete(nilInBlock, lIdent.Name)
					}
				}
			}
		})
		v.ifIf(bStmt, func(ifStmt *ast.IfStmt) {
			assignedInIfStmt, nilledInIfStmt := v.analysisIf(ifStmt, make(map[string]struct{}), make(map[string]struct{}), true)
			for assigned := range assignedInIfStmt {
				assignInBlock[assigned] = empty
			}
			for nilled := range nilledInIfStmt {
				nilInBlock[nilled] = empty
			}
		})
	}
	return assignInBlock, nilInBlock
}

func (v *testVisitor) analysisIf(ifStmt *ast.IfStmt, assigned map[string]struct{}, nilled map[string]struct{}, init bool) (map[string]struct{}, map[string]struct{}) {
	if ifStmt.Else == nil {
		assigned = make(map[string]struct{})
	}
	assignInBranch, nilInBranch := v.analysisAssignInBlock(ifStmt.Body.List)
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
			assignInBranch, nilInBranch := v.analysisAssignInBlock(elseBlock.List)
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

func (v *testVisitor) isNil(e ast.Expr) bool {
	return v.info.Types[e].Type == types.Typ[types.UntypedNil]
}

func main() {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", input, 0)
	if err != nil {
		panic(err)
	}
	info := types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	var conf types.Config
	_, err = conf.Check("fib", fset, []*ast.File{file}, &info)
	if err != nil {
		panic(err)
	}
	ast.Walk(&testVisitor{info}, file)
	// printer.Fprint(os.Stdout, fset, file)
}

const input = `
package main

type TestErr struct {
	Msg string
}

func (t *TestErr) Error() string {
	return "test err"
}

func b() (error, error) {
    var a *TestErr = nil
    var b *TestErr = &TestErr{}
    a = &TestErr{}
    a = nil
    if true {
       if true {
          b = nil
          a = &TestErr{}
          b = &TestErr{}
          b = nil
       } else {
       	  a = &TestErr{}
       }
    } else if true {
       a = &TestErr{}
    } else {
       a = &TestErr{}
    }
	return a, b
}

func main() {
    e1, _ := b()
	if e1 == nil {
		panic("b == nil")
	} else {
		panic("b != nil")
	}
}
`
