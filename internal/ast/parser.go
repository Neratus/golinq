package ast

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

func parseQueryCall(call *ast.CallExpr, fset *token.FileSet, query *QuerySpec) error {
	if call == nil || call.Fun == nil {
		return nil
	}
	if idx, ok := call.Fun.(*ast.IndexExpr); ok {
		sel, ok := idx.X.(*ast.SelectorExpr)
		if !ok {
			return fmt.Errorf("expected SelectorExpr inside IndexExpr, got %T", idx.X)
		}
		if sel.Sel.Name != "Select" {
			return fmt.Errorf("expected Select method, got %s", sel.Sel.Name)
		}
		return parseSelectCall(call, fset, query)
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return fmt.Errorf("expected SelectorExpr, got %T", call.Fun)
	}
	func_name := sel.Sel.Name
	switch func_name {
	case "Select":
		return parseSelectCall(call, fset, query)
	case "ToList":
		query.Method = "ToList"
		nextCall, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return fmt.Errorf("expected call before ToList, got %T", sel.X)
		}
		return parseQueryCall(nextCall, fset, query)
	case "First":
		query.Method = "First"
		nextCall, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return fmt.Errorf("expected call before First, got %T", sel.X)
		}
		return parseQueryCall(nextCall, fset, query)
	case "Where":
		if len(call.Args) < 1 {
			return fmt.Errorf("Where expects at least 1 argument (predicate name)")
		}
		var predName string
		switch arg := call.Args[0].(type) {
		case *ast.Ident:
			predName = arg.Name
		case *ast.SelectorExpr:
			predName = exprToString(fset, arg)
		default:
			return fmt.Errorf("Where argument must be an identifier or selector, got %T", call.Args[0])
		}
		var args []ast.Expr
		for i := 1; i < len(call.Args); i++ {
			args = append(args, call.Args[i])
		}
		pred := QueryStep{
			Type:          StepWhere,
			Predicate:     predName,
			PredicateArgs: args,
		}
		query.Steps = append([]QueryStep{pred}, query.Steps...)
		nextCall, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return fmt.Errorf("expected call before %s, got %T", func_name, sel.X)
		}
		return parseQueryCall(nextCall, fset, query)
	case "Limit":
		if len(call.Args) != 1 {
			return fmt.Errorf("Limit expects exactly 1 argument, got %d", len(call.Args))
		}
		lim, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lim.Kind != token.INT {
			return fmt.Errorf("Limit argument must be an identifier, got %T", call.Args[0])
		}
		limitVal, err := strconv.Atoi(lim.Value)
		if err != nil {
			return fmt.Errorf("invalid limit value: %v", err)
		}
		query.LimitVal = limitVal
		nextCall, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return fmt.Errorf("expected call before %s, got %T", func_name, sel.X)
		}
		return parseQueryCall(nextCall, fset, query)
	case "Offset":
		if len(call.Args) != 1 {
			return fmt.Errorf("Offset expects exactly 1 argument, got %d", len(call.Args))
		}
		off, ok := call.Args[0].(*ast.BasicLit)
		if !ok || off.Kind != token.INT {
			return fmt.Errorf("Offset argument must be an identifier, got %T", call.Args[0])
		}
		offVal, err := strconv.Atoi(off.Value)
		if err != nil {
			return fmt.Errorf("invalid offset value: %v", err)
		}
		query.OffsetVal = offVal
		nextCall, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return fmt.Errorf("expected call before %s, got %T", func_name, sel.X)
		}
		return parseQueryCall(nextCall, fset, query)
	case "OrderBy":
		if len(call.Args) < 1 || len(call.Args) > 2 {
			return fmt.Errorf("OrderBy expects 1 or 2 arguments, got %d", len(call.Args))
		}
		field, ok := call.Args[0].(*ast.BasicLit)
		if !ok || field.Kind != token.STRING {
			return fmt.Errorf("OrderBy argument must be an identifier, got %T", call.Args[0])
		}
		name := strings.Trim(field.Value, `"`)

		desc := false
		if len(call.Args) == 2 {
			switch arg := call.Args[1].(type) {
			case *ast.Ident:
				desc = arg.Name == "true"
			case *ast.BasicLit:
				if arg.Kind == token.STRING {
					val := strings.Trim(arg.Value, `"`)
					desc = val == "true"
				} else if arg.Kind == token.IDENT {
					desc = arg.Value == "true"
				} else {
					return fmt.Errorf("OrderBy second argument must be a boolean, got %s", arg.Value)
				}
			default:
				return fmt.Errorf("OrderBy second argument must be a boolean literal or identifier")
			}
		}
		query.OrderBy = &OrderByClause{
			Field: name,
			Desc:  desc,
		}
		nextCall, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return fmt.Errorf("expected call before %s, got %T", func_name, sel.X)
		}
		return parseQueryCall(nextCall, fset, query)
	case "Join":
		if len(call.Args) != 1 {
			return fmt.Errorf("Join expects exactly 1 argument, got %d", len(call.Args))
		}
		var joinName string
		switch arg := call.Args[0].(type) {
		case *ast.Ident:
			joinName = arg.Name
		case *ast.SelectorExpr:
			joinName = exprToString(fset, arg)
		default:
			return fmt.Errorf("Join argument must be an identifier or selector, got %T", call.Args[0])
		}
		join := QueryStep{
			Type:    StepJoin,
			Join:    joinName,
			JoinRef: nil,
		}

		query.Steps = append([]QueryStep{join}, query.Steps...)
		nextCall, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return fmt.Errorf("expected call before %s, got %T", func_name, sel.X)
		}
		return parseQueryCall(nextCall, fset, query)
	default:
		return fmt.Errorf("unexpected method %q in query chain", func_name)
	}
}

func parseSelectCall(call *ast.CallExpr, fset *token.FileSet, query *QuerySpec) error {
	idx, ok := call.Fun.(*ast.IndexExpr)
	if !ok {
		return fmt.Errorf("Select call must be generic: Select[T]")
	}
	sel, ok := idx.X.(*ast.SelectorExpr)
	if !ok {
		return fmt.Errorf("Select call must be of form pkg.Select[T]")
	}
	if sel.Sel.Name != "Select" {
		return fmt.Errorf("expected Select method, got %s", sel.Sel.Name)
	}

	typeName := exprToString(fset, idx.Index)
	if typeName == "" {
		return fmt.Errorf("Select call has empty type argument")
	}
	parts := strings.SplitN(typeName, ".", 2)
	if len(parts) == 2 {
		query.ModelImportAlias = parts[0]
		query.StructName = parts[1]
	} else {
		query.StructName = typeName
	}
	query.StructName = typeName

	if len(call.Args) < 2 {
		return nil
	}
	for i, arg := range call.Args {
		if i == 0 {
			continue
		}
		lit, ok := arg.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return fmt.Errorf("Select column argument must be a string literal")
		}
		colName := strings.Trim(lit.Value, `"`)
		query.SelectCols = append(query.SelectCols, colName)
	}

	return nil
}

func ParseFiles(f string, res *ProjectQueries) error {
	key_num := 0
	var alias string

	models := make(map[string]*ModelMeta)
	predicates := make(map[string]*PredicateMeta)
	joins := make(map[string]*JoinMeta)
	queryCalls := make(map[string]*QuerySpec)
	funcMeta := make(map[string]*FuncDeclMeta)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, f, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	imports := make(map[string]string)
	for _, imp := range file.Imports {
		var cur_alias string
		path := strings.Trim(imp.Path.Value, `"`)
		if imp.Name != nil {
			cur_alias = imp.Name.Name
		} else {
			cur_alias = filepath.Base(path)
		}
		imports[cur_alias] = path
		if strings.HasSuffix(path, "golinq") {
			alias = cur_alias
		}
	}
	res.FileImports[f] = &FileImports{
		FilePath:    f,
		PackageName: file.Name.Name,
		Imports:     imports,
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			if x.Tok != token.TYPE || x.Doc == nil || !strings.Contains(strings.ReplaceAll(x.Doc.Text(), " ", ""), "golinq:model") {
				return true
			}
			for _, spec := range x.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				x, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				modelMeta := ModelMeta{
					PackageName: file.Name.Name,
					PackagePath: file.Name.Name + "." + typeSpec.Name.Name,
					StructName:  typeSpec.Name.Name,
					TableName:   convertToSnakeCase(typeSpec.Name.Name),
					FilePath:    f,
				}

				fields := []StructField{}
				for _, field := range x.Fields.List {
					if len(field.Names) == 0 {
						continue
					}
					fieldName := field.Names[0].Name
					varName := convertToSnakeCase(fieldName)
					fieldTypeStr := exprToString(fset, field.Type)
					isPrimary := false
					skip := false
					if field.Tag != nil {
						rawTag := field.Tag.Value
						stripped := rawTag[1 : len(rawTag)-1]
						tag := reflect.StructTag(stripped).Get("golinq")

						parts := strings.Split(tag, ",")
						for _, part := range parts {
							if part == "primary_key" {
								isPrimary = true
							} else if part == "skip" {
								skip = true
							} else if after, ok0 := strings.CutPrefix(part, "column="); ok0 {
								varName = after
							}

						}
					}
					if skip {
						continue
					}
					field := StructField{
						FieldName:  fieldName,
						MappingSQL: varName,
						FieldType:  fieldTypeStr,
						IsPrimary:  isPrimary,
						Skip:       skip,
					}
					fields = append(fields, field)

				}
				modelMeta.Fields = fields
				fullKey := modelMeta.PackageName + "." + modelMeta.StructName
				models[fullKey] = &modelMeta
			}
		case *ast.FuncDecl:
			if x.Type.Results == nil || len(x.Type.Results.List) != 1 || exprToString(fset, x.Type.Results.List[0].Type) != "bool" {
				return true
			}
			fullFuncKey := file.Name.Name + "." + x.Name.Name
			funcMeta[fullFuncKey] = &FuncDeclMeta{
				Name:        x.Name.Name,
				PackageName: file.Name.Name,
				PackagePath: "",
				FuncDecl:    x,
				IsUsed:      false,
			}
		case *ast.ValueSpec:
			if len(x.Names) == 0 || len(x.Values) == 0 {
				return true
			}
			if len(x.Names) != 1 || len(x.Values) != 1 {
				return true
			}
			name := x.Names
			for _, expr := range x.Values {
				call, ok := expr.(*ast.CallExpr)
				if !ok {
					// Не вызов функции – просто игнорируем
					continue
				}
				var qc QuerySpec
				err := parseQueryCall(call, fset, &qc)
				if err == nil && qc.StructName != "" {
					qc.PackageName = file.Name.Name
					qc.PackagePath = f
					key := fmt.Sprintf("%d&%s@%d", key_num, file.Name.Name, len(queryCalls))
					key_num++
					queryCalls[key] = &qc
					continue
				}
				var pkgIdent *ast.Ident
				var funcName string
				var genericType string

				switch fun := call.Fun.(type) {
				case *ast.IndexExpr:
					sel, ok := fun.X.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					pkgIdent, ok = sel.X.(*ast.Ident)
					if !ok || pkgIdent == nil {
						continue
					}
					funcName = sel.Sel.Name
					genericType = exprToString(fset, fun.Index)

				case *ast.SelectorExpr:
					pkgIdent, ok = fun.X.(*ast.Ident)
					if !ok || pkgIdent == nil {
						continue
					}
					funcName = fun.Sel.Name

				default:
					continue
				}
				if pkgIdent.Name != alias {
					continue
				}

				switch funcName {
				case "Predicate":
					if pkgIdent.Name != alias {
						return true
					}
					modelTypeStr := genericType

					funcLit, ok := call.Args[0].(*ast.FuncLit)
					ident := ""
					if !ok {
						body, ok := call.Args[0].(*ast.Ident)
						if !ok {
							return true
						}
						funcLit = nil
						ident = body.Name
					}
					arg := []*PredicateArg{}
					if funcLit != nil && funcLit.Type != nil && funcLit.Type.Params != nil && len(funcLit.Type.Params.List) > 0 {
						for _, param := range funcLit.Type.Params.List {
							for _, name := range param.Names {
								arg = append(arg, &PredicateArg{
									Name:        name.Name,
									TypeName:    exprToString(fset, param.Type),
									PackageName: file.Name.Name,
									Declared:    false,
								})
							}
						}
					}
					varName := name[0].Name
					predMeta := PredicateMeta{
						PredicateName:  name[0].String(),
						PackageName:    file.Name.Name,
						PackagePath:    f,
						GlobalVarName:  varName,
						ModelType:      modelTypeStr,
						Args:           arg,
						FuncBody:       funcLit,
						LinkedFuncName: &ident,
					}
					fullPredKey := predMeta.PackageName + "." + predMeta.PredicateName
					predicates[fullPredKey] = &predMeta
				case "Join":
					if len(call.Args) != 3 || pkgIdent.Name != alias {
						return true
					}
					leftModelExpr := call.Args[0]
					rightModelExpr := call.Args[1]
					leftModelStr := extractTypeName(fset, leftModelExpr)
					rightModelStr := extractTypeName(fset, rightModelExpr)

					funcLit, ok := call.Args[2].(*ast.FuncLit)
					ident := ""
					if !ok {
						if id, ok := call.Args[2].(*ast.Ident); ok {
							ident = id.Name
						} else {
							return true
						}
						funcLit = nil
					}

					left := []PredicateArg{}
					right := []PredicateArg{}
					if funcLit != nil && funcLit.Type != nil && funcLit.Type.Params != nil && len(funcLit.Type.Params.List) > 0 {
						for i, param := range funcLit.Type.Params.List {
							for _, name := range param.Names {
								if i%2 == 0 {
									left = append(left, PredicateArg{
										Name:        name.Name,
										TypeName:    exprToString(fset, param.Type),
										PackageName: file.Name.Name,
										Declared:    false,
									})
								} else {
									right = append(right, PredicateArg{
										Name:        name.Name,
										TypeName:    exprToString(fset, param.Type),
										PackageName: file.Name.Name,
										Declared:    false,
									})
								}
							}
						}
					}
					varName := name[0].Name
					if (len(left) == 0 || len(right) == 0) && funcLit != nil {
						return true
					} else if funcLit == nil {
						joinMeta := &JoinMeta{
							JoinName:       name[0].String(),
							PackageName:    file.Name.Name,
							PackagePath:    f,
							GlobalVarName:  varName,
							LeftArg:        PredicateArg{},
							LeftModelType:  leftModelStr,
							RightArg:       PredicateArg{},
							RightModelType: rightModelStr,
							OnFunc:         funcLit,
							JoinType:       "INNER",
							LinkedFuncName: &ident,
						}
						fullJoinKey := joinMeta.PackageName + "." + joinMeta.JoinName
						joins[fullJoinKey] = joinMeta
					} else {
						joinMeta := &JoinMeta{
							JoinName:       name[0].String(),
							PackageName:    file.Name.Name,
							PackagePath:    f,
							GlobalVarName:  varName,
							LeftArg:        left[0],
							LeftModelType:  leftModelStr,
							RightArg:       right[0],
							RightModelType: rightModelStr,
							OnFunc:         funcLit,
							JoinType:       "INNER",
							LinkedFuncName: &ident,
						}
						fullJoinKey := joinMeta.PackageName + "." + joinMeta.JoinName
						joins[fullJoinKey] = joinMeta
					}

				default:
					return true
				}

			}
		case *ast.AssignStmt:
			for _, rhs := range x.Rhs {
				if call, ok := rhs.(*ast.CallExpr); ok {
					var qc QuerySpec
					if err := parseQueryCall(call, fset, &qc); err == nil && qc.StructName != "" {
						qc.PackageName = file.Name.Name
						qc.PackagePath = f
						key := fmt.Sprintf("%d&%s@%d", key_num, file.Name.Name, len(queryCalls))
						key_num++
						queryCalls[key] = &qc
					} else if err != nil {
						fmt.Printf("DEBUG: parseQueryCall error in %s: %v\n", f, err)
					}
				}
			}
		case *ast.ExprStmt:
			if call, ok := x.X.(*ast.CallExpr); ok {
				var qc QuerySpec
				if err := parseQueryCall(call, fset, &qc); err == nil && qc.StructName != "" {
					qc.PackageName = file.Name.Name
					qc.PackagePath = f
					key := fmt.Sprintf("%d&%s@%d", key_num, file.Name.Name, len(queryCalls))
					key_num++
					queryCalls[key] = &qc
				} else if err != nil {
					fmt.Printf("DEBUG: parseQueryCall error in %s: %v\n", f, err)
				}
			}
		}

		return true
	})
	for _, q := range queryCalls {
		if q.ModelImportAlias != "" && q.PackagePath != "" {
			if fileImports, ok := res.FileImports[q.PackagePath]; ok {
				if path, ok := fileImports.Imports[q.ModelImportAlias]; ok {
					q.ModelImportPath = path
				}
			}
		}
	}
	maps.Copy(res.Models, models)
	maps.Copy(res.Predicates, predicates)
	maps.Copy(res.Joins, joins)
	maps.Copy(res.QueryCalls, queryCalls)
	maps.Copy(res.FuncDecls, funcMeta)
	return nil
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
