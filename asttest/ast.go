package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
)

type testVisitor struct {
	info types.Info
}

func (v *testVisitor) Visit(node ast.Node) (w ast.Visitor) {
	switch t := node.(type) {
	case *ast.FuncDecl:
		if t.Type.Results != nil {
			results := t.Type.Results.List
			for _, result := range results {
				tv := v.info.Types[result.Type]
				fmt.Println(tv.Type, types.IsInterface(tv.Type))
				if types.IsInterface(tv.Type) {
					for _, stmt := range t.Body.List {
						if rs, ok := stmt.(*ast.ReturnStmt); ok {
							fmt.Println(rs.Results[0])
						}
					}
				}
			}
		}
	}
	return v
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
	pkg, err := conf.Check("fib", fset, []*ast.File{file}, &info)
	if err != nil {
		panic(err)
	}
	fmt.Println(pkg.Name())
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

func test() *TestErr {
	return nil
}

func b() error {
	return test()
}

func main() {
	if b() == nil {
		panic("b == nil")
	} else {
		panic("b != nil")
	}
}
`