package ast

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

func isUpperCase(s string) bool {
	return s == strings.ToUpper(s)
}

func nodeToString(fset *token.FileSet, n ast.Node) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, n); err != nil {
		return fmt.Sprintf("%T", n)
	}
	return buf.String()
}

func typeToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return typeToString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + typeToString(e.X)
	case *ast.ArrayType:
		return "[]" + typeToString(e.Elt)
	default:
		return fmt.Sprintf("%T", e)
	}
}

func extractTypeName(fset *token.FileSet, expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprToString(fset, e)
	case *ast.CompositeLit:
		return extractTypeName(fset, e.Type)
	default:
		return exprToString(fset, e)
	}
}

func exprToString(fset *token.FileSet, expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	case *ast.CallExpr:
		return nodeToString(fset, e)
	default:
		return nodeToString(fset, e)
	}
}

func convertToSnakeCase(str string) string {
	res := ""
	for i, rune := range str {
		symb := string(rune)
		if i == 0 {
			res += strings.ToLower(symb)
		} else if isUpperCase(symb) {
			res += "_" + strings.ToLower(symb)
		} else {
			res += symb
		}
	}
	return res
}

func isBasicLitCompatible(lit *ast.BasicLit, expectedType string) bool {
	switch expectedType {
	case "int", "int32", "int64", "float32", "float64":
		return lit.Kind == token.INT || lit.Kind == token.FLOAT
	case "string":
		return lit.Kind == token.STRING
	case "bool":
		return lit.Kind == token.IDENT && (lit.Value == "true" || lit.Value == "false")
	default:
		return false
	}
}

func ParseDir(root string) (*ProjectQueries, error) {
	res := &ProjectQueries{
		Models:      make(map[string]*ModelMeta),
		Predicates:  make(map[string]*PredicateMeta),
		Joins:       make(map[string]*JoinMeta),
		QueryCalls:  make(map[string]*QuerySpec),
		FileImports: make(map[string]*FileImports),
		FuncDecls:   make(map[string]*FuncDeclMeta),
	}
	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if err := ParseFiles(path, res); err != nil {
			return fmt.Errorf("error processing file %q: %w", path, err)
		}
		return nil
	})
	return res, err
}
