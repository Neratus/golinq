package golinq

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	go_ast "github.com/Neratus/golinq/internal/ast"
)

var param = 0

func buildExpr(expr ast.Expr, paramNames map[string]int, args []ast.Expr, paramToAlias map[string]string, models map[string]*go_ast.ModelMeta, isFunc bool, imports map[string]string) (*ConditionNode, error) {
	switch n := expr.(type) {
	case *ast.Ident:
		if n.Name == "true" || n.Name == "false" {
			return &ConditionNode{
				Type:  Const,
				Value: n.Name == "true",
			}, nil
		}
		if idx, ok := paramNames[n.Name]; ok {
			if idx == 0 {
				return nil, fmt.Errorf("model parameter %s cannot be used alone, expected field access", n.Name)
			}
			if idx-1 >= len(args) {
				return nil, fmt.Errorf("missing argument for parameter %s (index %d)", n.Name, idx)
			}
			return buildExpr(args[idx-1], paramNames, args, paramToAlias, models, isFunc, imports)
		}
		param += 1
		return &ConditionNode{
			Type:       Param,
			Value:      n.Name,
			Children:   nil,
			ParamIndex: param,
		}, nil
	case *ast.BasicLit:
		var val any
		switch n.Kind {
		case token.STRING:
			val = strings.Trim(n.Value, `"`)
		case token.INT:
			val, _ = strconv.Atoi(n.Value)
		case token.FLOAT:
			val, _ = strconv.ParseFloat(n.Value, 64)
		default:
			val = n.Value
		}
		return &ConditionNode{
			Type:     Const,
			Value:    val,
			Children: nil,
		}, nil
	case *ast.BinaryExpr:
		leftNode, err := buildExpr(n.X, paramNames, args, paramToAlias, models, isFunc, imports)
		if err != nil {
			return nil, err
		}
		rightNode, err := buildExpr(n.Y, paramNames, args, paramToAlias, models, isFunc, imports)
		if err != nil {
			return nil, err
		}
		var opVal string
		var opType NodeType
		switch n.Op {
		case token.EQL:
			opVal = "EQ"
			opType = Cmp
		case token.NEQ:
			opVal = "NEQ"
			opType = Cmp
		case token.GTR:
			opVal = "GT"
			opType = Cmp
		case token.GEQ:
			opVal = "GE"
			opType = Cmp
		case token.LSS:
			opVal = "LT"
			opType = Cmp
		case token.LEQ:
			opVal = "LE"
			opType = Cmp
		case token.LAND:
			opVal = "AND"
			opType = And
		case token.LOR:
			opVal = "OR"
			opType = Or
		default:
			return nil, fmt.Errorf("unsupported binary operator: %v", n.Op)
		}
		return &ConditionNode{
			Type:     opType,
			Value:    opVal,
			Children: []*ConditionNode{leftNode, rightNode},
		}, nil
	case *ast.SelectorExpr:
		if ident, ok := n.X.(*ast.Ident); ok {
			if _, ok := paramNames[ident.Name]; ok {
				var sqlName string
				alias, ok := paramToAlias[ident.Name]
				if !ok {
					return nil, fmt.Errorf("no alias for model parameter %s", ident.Name)
				}
				for _, model := range models {
					if model.StructName == alias {
						for _, f := range model.Fields {
							if f.FieldName == n.Sel.Name {
								if !isFunc {
									sqlName = f.MappingSQL
									break
								} else {
									alias = model.StructName
									sqlName = f.FieldName
									break
								}
							}
						}
						break
					}
				}
				return &ConditionNode{
					Type:     Field,
					Value:    alias + "." + sqlName,
					Children: nil,
				}, nil
			}
		}
		return nil, fmt.Errorf("invalid selector: %s (expected model parameter)", n.Sel.Name)
	case *ast.ParenExpr:
		return buildExpr(n.X, paramNames, args, paramToAlias, models, isFunc, imports)
	case *ast.UnaryExpr:
		exp := n.Op
		if exp == token.NOT {
			node, err := buildExpr(n.X, paramNames, args, paramToAlias, models, isFunc, imports)
			if err != nil {
				return nil, err
			}
			return &ConditionNode{
				Type:     Not,
				Value:    "NOT",
				Children: []*ConditionNode{node},
			}, nil
		}
		return nil, fmt.Errorf("unsupported unary operator: %v", n.Op)
	case *ast.CallExpr:
		if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
			if pkgIdent, ok := sel.X.(*ast.Ident); ok {
				if importPath, ok := imports[pkgIdent.Name]; ok && importPath == "strings" {
					funcName := sel.Sel.Name
					switch funcName {
					case "Contains", "HasPrefix", "HasSuffix":
						var children []*ConditionNode
						for _, arg := range n.Args {
							child, err := buildExpr(arg, paramNames, args, paramToAlias, models, isFunc, imports)
							if err != nil {
								return nil, err
							}
							children = append(children, child)
						}
						if funcName == "Contains" {
							return &ConditionNode{Type: Like, Children: children}, nil
						} else {
							return &ConditionNode{Type: Like, Value: funcName, Children: children}, nil
						}
					default:
						return nil, fmt.Errorf("unsupported strings function: %s", funcName)
					}
				}
			}
		}
		var children []*ConditionNode
		var funcIdent string
		switch fun := n.Fun.(type) {
		case *ast.Ident:
			funcIdent = fun.Name
		case *ast.SelectorExpr:
			var pkg string
			if x, ok := fun.X.(*ast.Ident); ok {
				pkg = x.Name
			} else {
				pkg = fmt.Sprintf("%T", n.Fun)
			}
			funcIdent = pkg + "." + fun.Sel.Name
		default:
			return nil, fmt.Errorf("unsupported function call type: %T", n.Fun)
		}
		for _, arg := range n.Args {
			node, err := buildExpr(arg, paramNames, args, paramToAlias, models, true, imports)
			if err != nil {
				return nil, err
			}
			children = append(children, node)
		}
		return &ConditionNode{
			Type:     Func,
			Value:    funcIdent,
			Children: children,
		}, nil
	case *ast.StarExpr:
		return buildExpr(n.X, paramNames, args, paramToAlias, models, isFunc, imports)
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", n)
	}
}

func constructJoinNode(join *go_ast.JoinMeta, leftAlias, rightAlias string, models map[string]*go_ast.ModelMeta, imports map[string]string) (*ConditionNode, error) {
	var funcBody *ast.BlockStmt
	var returnExpr *ast.ReturnStmt
	paramNames := make(map[string]int)
	if join.OnFunc == nil && join.FuncDeclRef == nil {
		return nil, fmt.Errorf("predicate %s has no function body", join.JoinName)
	}
	if join.OnFunc != nil {
		funcBody = join.OnFunc.Body
	} else {
		funcBody = join.FuncDeclRef.Body
	}
	for _, stmt := range funcBody.List {
		if rs, ok := stmt.(*ast.ReturnStmt); ok {
			returnExpr = rs
			break
		}
	}
	if returnExpr == nil {
		return nil, fmt.Errorf("predicate %s has no return statement", join.JoinName)
	}
	if len(returnExpr.Results) != 1 {
		return nil, fmt.Errorf("predicate %s must return exactly one expression", join.JoinName)
	}
	if join.LeftArg.Name != "" {
		paramNames[join.LeftArg.Name] = 0
	}
	if join.RightArg.Name != "" {
		paramNames[join.RightArg.Name] = 1
	}
	paramToAlias := map[string]string{
		join.LeftArg.Name:  leftAlias,
		join.RightArg.Name: rightAlias,
	}
	return buildExpr(returnExpr.Results[0], paramNames, nil, paramToAlias, models, false, imports)
}

func constructPredNode(pred *go_ast.PredicateMeta, args []ast.Expr, currentAlias string, models map[string]*go_ast.ModelMeta, imports map[string]string) (*ConditionNode, error) {
	var funcBody *ast.BlockStmt
	var returnExpr *ast.ReturnStmt
	paramNames := make(map[string]int)
	paramToAlias := make(map[string]string)
	if pred.FuncBody == nil && pred.FuncDeclRef == nil {
		return nil, fmt.Errorf("predicate %s has no function body", pred.PredicateName)
	}
	if pred.FuncBody != nil {
		funcBody = pred.FuncBody.Body
	} else {
		funcBody = pred.FuncDeclRef.Body
	}
	for _, stmt := range funcBody.List {
		if rs, ok := stmt.(*ast.ReturnStmt); ok {
			returnExpr = rs
			break
		}
	}
	if returnExpr == nil {
		return nil, fmt.Errorf("predicate %s has no return statement", pred.PredicateName)
	}
	if len(returnExpr.Results) != 1 {
		return nil, fmt.Errorf("predicate %s must return exactly one expression", pred.PredicateName)
	}
	for i, arg := range pred.Args {
		paramNames[arg.Name] = i
	}
	if len(pred.Args) > 0 {
		paramToAlias[pred.Args[0].Name] = currentAlias
	}
	return buildExpr(returnExpr.Results[0], paramNames, args, paramToAlias, models, false, imports)
}

// extractShortName возвращает имя структуры без пакета
func extractShortName(full string) string {
	if idx := strings.LastIndex(full, "."); idx != -1 {
		return full[idx+1:]
	}
	return full
}

// getModelByFullName ищет модель по полному имени (с пакетом) или по короткому имени
func getModelByFullName(fullName string, models map[string]*go_ast.ModelMeta) (*go_ast.ModelMeta, error) {
	if model, ok := models[fullName]; ok {
		return model, nil
	}
	short := extractShortName(fullName)
	for _, m := range models {
		if m.StructName == short {
			return m, nil
		}
	}
	return nil, fmt.Errorf("model %s not found", fullName)
}

func BuildSelectAstTree(res *go_ast.QuerySpec, models map[string]*go_ast.ModelMeta, fileImports map[string]*go_ast.FileImports) (*SelectQueryAST, error) {
	param = 0
	var whereNode []*ConditionNode
	var joinNodes []JoinNode
	var imports map[string]string
	if fi, ok := fileImports[res.PackagePath]; ok {
		imports = fi.Imports
	}

	// Построить маппинг алиас таблицы -> модель
	aliasToModel := make(map[string]*go_ast.ModelMeta)
	currentModel, err := getModelByFullName(res.StructName, models)
	if err != nil {
		return nil, err
	}
	currentAlias := currentModel.StructName
	aliasToModel[currentAlias] = currentModel

	// Зарегистрировать все модели из JOIN
	for _, step := range res.Steps {
		if step.Type == go_ast.StepJoin {
			join := step.JoinRef
			leftModel, err := getModelByFullName(join.LeftModelType, models)
			if err != nil {
				return nil, err
			}
			rightModel, err := getModelByFullName(join.RightModelType, models)
			if err != nil {
				return nil, err
			}
			leftAlias := leftModel.StructName
			rightAlias := rightModel.StructName
			if _, ok := aliasToModel[leftAlias]; !ok {
				aliasToModel[leftAlias] = leftModel
			}
			if _, ok := aliasToModel[rightAlias]; !ok {
				aliasToModel[rightAlias] = rightModel
			}
		}
	}

	// Построить узлы JOIN (с условиями ON) и одновременно обновлять текущий алиас для WHERE
	currentAliasForJoins := currentModel.StructName
	for _, step := range res.Steps {
		if step.Type == go_ast.StepJoin {
			join := step.JoinRef
			rightModel, err := getModelByFullName(join.RightModelType, models)
			if err != nil {
				return nil, err
			}
			rightAlias := rightModel.StructName
			node, err := constructJoinNode(join, currentAliasForJoins, rightAlias, models, imports)
			if err != nil {
				return nil, err
			}
			joinNodes = append(joinNodes, JoinNode{
				Type: join.JoinType,
				Left: nil,
				Right: Relation{
					Name:  rightModel.TableName,
					Alias: rightAlias,
				},
				On: node,
			})
			currentAliasForJoins = rightAlias
		}
	}

	// Построить узел WHERE, обновляя текущий алиас при проходе по шагам
	curAlias := currentModel.StructName
	for _, step := range res.Steps {
		if step.Type == go_ast.StepWhere {
			node, err := constructPredNode(step.PredicateRef, step.PredicateArgs, curAlias, models, imports)
			if err != nil {
				return nil, err
			}
			whereNode = append(whereNode, node)
		} else if step.Type == go_ast.StepJoin {
			join := step.JoinRef
			rightModel, err := getModelByFullName(join.RightModelType, models)
			if err != nil {
				return nil, err
			}
			curAlias = rightModel.StructName
		}
	}
	predNode := &ConditionNode{
		Type:     And,
		Children: whereNode,
	}

	var queryMethod QueryMethod
	switch res.Method {
	case "ToList":
		queryMethod = ToList
	case "First":
		queryMethod = First
	default:
		queryMethod = ToList
	}

	// Построить SelectFields на основе res.SelectCols (теперь []SelectFieldSpec)
	var selectFields []SelectField
	if len(res.SelectCols) != 0 {
		for _, selSpec := range res.SelectCols {
			alias := selSpec.TableAlias
			colName := selSpec.ColumnName
			if alias == "" {
				alias = curAlias // текущий алиас после всех шагов
			}
			model, ok := aliasToModel[alias]
			if !ok {
				return nil, fmt.Errorf("unknown table alias %q in SELECT", alias)
			}
			var fieldMeta *go_ast.StructField
			for i, f := range model.Fields {
				if f.FieldName == colName {
					fieldMeta = &model.Fields[i]
					break
				}
			}
			if fieldMeta == nil {
				return nil, fmt.Errorf("column %q not found in model %s", colName, model.StructName)
			}
			selectFields = append(selectFields, SelectField{
				TableAlias: alias,
				ColumnName: fieldMeta.MappingSQL,
				GoType:     fieldMeta.FieldType,
			})
		}
	} else {
		// Если SelectCols не указаны, выбираем все поля из текущей модели
		model, ok := aliasToModel[curAlias]
		if !ok {
			return nil, fmt.Errorf("current model alias %s not found in alias map", curAlias)
		}
		for _, f := range model.Fields {
			selectFields = append(selectFields, SelectField{
				TableAlias: curAlias,
				ColumnName: f.MappingSQL,
				GoType:     f.FieldType,
			})
		}
	}

	var orderByClause *OrderByClause
	if res.OrderBy != nil {
		// Разбираем "Country.Name" на алиас и имя поля
		parts := strings.SplitN(res.OrderBy.Field, ".", 2)
		var alias, fieldName string
		if len(parts) == 2 {
			alias = parts[0]
			fieldName = parts[1]
		} else {
			// если нет точки, используем текущий алиас
			alias = curAlias
			fieldName = res.OrderBy.Field
		}
		model, ok := aliasToModel[alias]
		if !ok {
			return nil, fmt.Errorf("unknown table alias %q in OrderBy", alias)
		}
		var mapping string
		for _, field := range model.Fields {
			if field.FieldName == fieldName {
				mapping = field.MappingSQL
				break
			}
		}
		if mapping == "" {
			return nil, fmt.Errorf("order by field %s not found in model %s", fieldName, model.StructName)
		}
		orderByClause = &OrderByClause{
			TableAlias: alias,
			Field:      fieldName,
			MappingSQL: mapping,
			Desc:       res.OrderBy.Desc,
		}
	}

	query := &SelectQueryAST{
		SelectFields: selectFields,
		From: Relation{
			Name:  currentModel.TableName,
			Alias: currentModel.StructName,
		},
		Joins:   joinNodes,
		Where:   predNode,
		OrderBy: orderByClause,
		Limit:   int64(res.LimitVal),
		Offset:  int64(res.OffsetVal),
		Method:  queryMethod,
	}
	return query, nil
}
