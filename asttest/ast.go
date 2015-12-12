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

func (v *testVisitor) ifMethodReturnInterface(node ast.Node, funcDecl func(t *ast.FuncDecl)) {
	switch t := node.(type) {
	case *ast.FuncDecl:
		if t.Type.Results == nil {
			return
		}
		results := t.Type.Results.List
		for _, result := range results {
			tv := v.info.Types[result.Type]
			if !types.IsInterface(tv.Type) {
				continue
			}
			funcDecl(t)
		}
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
	v.ifMethodReturnInterface(node, func(t *ast.FuncDecl) {

		mayNilVars := make(map[string]struct{})

		for _, stmt := range t.Body.List {
			v.ifDeclVar(stmt, func(genDecl *ast.GenDecl) {
				for _, spec := range genDecl.Specs {
					if varSpec, ok := spec.(*ast.ValueSpec); ok {
						if _, ok := varSpec.Type.(*ast.StarExpr); !ok {
							continue
						}
						for i, name := range varSpec.Names {
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
							mayNilVars[lIdent.Name] = empty
						}
					}
				}
			})

			v.ifIf(stmt, func(ifStmt *ast.IfStmt) {
				assignInIf := make(map[string]struct{})
				if ifStmt.Else == nil {
					return
				}
				assignInBranch := make(map[string]struct{})
				for _, bStmt := range ifStmt.Body.List {
					v.ifAssign(bStmt, func(assign *ast.AssignStmt) {
						for i, lexp := range assign.Lhs {
							if lIdent, lok := lexp.(*ast.Ident); lok {
								isNil := v.isNil(assign.Rhs[i])
								if isNil {
									delete(assignInBranch, lIdent.Name)
								} else {
									assignInBranch[lIdent.Name] = empty
								}
							}
						}
					})
				}
				for name, v := range assignInBranch {
					assignInIf[name] = v
				}
				if elseBlock, ok := ifStmt.Else.(*ast.BlockStmt); ok {
					assignInBranch := make(map[string]struct{})
					for _, bStmt := range elseBlock.List {
						v.ifAssign(bStmt, func(assign *ast.AssignStmt) {
							for i, lexp := range assign.Lhs {
								if lIdent, lok := lexp.(*ast.Ident); lok {
									isNil := v.isNil(assign.Rhs[i])
									if isNil {
										delete(assignInBranch, lIdent.Name)
									} else {
										assignInBranch[lIdent.Name] = empty
									}
								}
							}
						})
					}
					for name, _ := range assignInIf {
						if _, ok := assignInBranch[name]; !ok {
							delete(assignInIf, name)
						}
					}
				}
				for assigned := range assignInIf {
					delete(mayNilVars, assigned)
				}
			})

			if retStmt, ok := stmt.(*ast.ReturnStmt); ok {
				for _, result := range retStmt.Results {
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

func b() error {
    var err *TestErr = nil
    err = &TestErr{}
    err = nil
    if true {
       err = &TestErr{}
    } else {
       err = &TestErr{}
    }
	return err
}

func main() {
	if b() == nil {
		panic("b == nil")
	} else {
		panic("b != nil")
	}
}
`
