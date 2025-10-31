// Package linter содержит анализатор на поиск выховов функций
// panic, log.Fata,os.Exit
package linter

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var MyAnalyzer = &analysis.Analyzer{
	Name: "myAnalyzer",
	Doc:  "check for panic call in all code, calls log.Fatal() and os.Exit() in non main function main package",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	packageName := ""
	for _, fast := range pass.Files {
		ast.Inspect(fast, func(node ast.Node) bool {

			switch x := node.(type) {
			case *ast.File:
				packageName = x.Name.Name
			case *ast.FuncDecl:
				if x.Name.Name == "main" && packageName == "main" {
					checkFuncDecl(pass, x, "main")
					return false
				}
			case *ast.SelectorExpr:
				if pkgName, ok := x.X.(*ast.Ident); ok {
					if (pkgName.Name == "log" && x.Sel.Name == "Fatal") || (pkgName.Name == "os" && x.Sel.Name == "Exit") {
						pass.Reportf(x.Pos(), "call log.Fatal/os.Exit in non main function of main package.")
					}
				}
			case *ast.CallExpr:
				if funcName, ok := x.Fun.(*ast.Ident); ok {
					if funcName.Name == "panic" {
						pass.Reportf(x.Pos(), "find panic call")
					}
				}
			}
			return true
		})
	}
	return nil, nil
}

func checkFuncDecl(pass *analysis.Pass, node ast.Node, funcName string) {
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			for _, item := range x.Rhs {
				if f, ok := item.(*ast.FuncLit); ok {
					checkFuncDecl(pass, f.Body, "")
					return false
				}
			}
		case *ast.SelectorExpr:
			if pkgName, ok := x.X.(*ast.Ident); ok {
				if ((pkgName.Name == "log" && x.Sel.Name == "Fatal") || (pkgName.Name == "os" && x.Sel.Name == "Exit")) && funcName != "main" {
					pass.Reportf(x.Pos(), "call log.Fatal/os.Exit in sub function main function main package")
				}
			}
		case *ast.CallExpr:
			if funcName, ok := x.Fun.(*ast.Ident); ok {
				if funcName.Name == "panic" {
					pass.Reportf(x.Pos(), "find panic call")
				}
			}

		}
		return true
	})
}
