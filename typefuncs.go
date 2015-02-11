package dispel

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

// walker adapts a function to satisfy the ast.Visitor interface.
// The function return whether the walk should proceed into the node's children.
type walker func(ast.Node) bool

func (w walker) Visit(node ast.Node) ast.Visitor {
	if w(node) {
		return w
	}
	return nil
}

// FindTypesFuncs parses the AST of files of a package, look for methods on types listed in typesNames.
// It returns a map of func names -> *ast.FuncDecl.
func FindTypesFuncs(path string, pkgName string, typeNames []string, excludeFiles []string) (map[string]*ast.FuncDecl, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, func(fi os.FileInfo) bool {
		for _, f := range excludeFiles {
			if fi.Name() == f {
				return false
			}
		}
		return true
	}, parser.DeclarationErrors)
	if err != nil {
		return nil, err
	}
	pkg, ok := pkgs[pkgName]
	if !ok {
		return nil, fmt.Errorf("%s: package not found in %q", pkgName, path)
	}

	funcDecls := make(map[string]*ast.FuncDecl)
	for _, astFile := range pkg.Files {
		ast.Walk(walker(func(node ast.Node) bool {
			switch v := node.(type) {
			case *ast.FuncDecl:
				if v.Recv != nil {
					// this is a method, find the type of the receiver
					field := v.Recv.List[0]
					var ident *ast.Ident
					// Look on value and pointer receivers
					switch t := field.Type.(type) {
					default:
						return true
					case *ast.StarExpr:
						ident, ok = t.X.(*ast.Ident)
						if !ok {
							return true
						}
					case *ast.Ident:
						ident = t
					}

					for _, typeName := range typeNames {
						if typeName == ident.Name {
							funcDecls[v.Name.String()] = v
							return true
						}
					}
				}
			}
			return true
		}), astFile)
	}
	return funcDecls, nil
}

// FindTypes parses the AST of files of a package, look for the types declared in those files, excluding those listed in excludeFiles.
// It returns a map of type names -> *ast.TypeSpec.
func FindTypes(path string, pkgName string, excludeFiles []string) (map[string]*ast.TypeSpec, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, func(fi os.FileInfo) bool {
		for _, f := range excludeFiles {
			if fi.Name() == f {
				return false
			}
		}
		return true
	}, parser.DeclarationErrors)
	if err != nil {
		return nil, err
	}
	pkg, ok := pkgs[pkgName]
	if !ok {
		return nil, fmt.Errorf("%s: package not found in %q", pkgName, path)
	}

	typeSpecs := make(map[string]*ast.TypeSpec)
	for _, astFile := range pkg.Files {
		ast.Walk(walker(func(node ast.Node) bool {
			switch v := node.(type) {
			case *ast.TypeSpec:
				typeSpecs[v.Name.String()] = v
			}
			return true
		}), astFile)
	}
	return typeSpecs, nil
}
