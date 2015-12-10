package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

type testVisitor struct {
}

func (v *testVisitor) Visit(node ast.Node) (w ast.Visitor) {
	switch t := node.(type) {
	case *ast.FuncDecl:
		fmt.Printf("%s\n", t.Name.Name)
	}
	return v
}

func main() {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test_data/test.go", nil, 0)
	if err != nil {
		panic(err)
	}
	ast.Walk(&testVisitor{}, file)
	// printer.Fprint(os.Stdout, fset, file)
}
