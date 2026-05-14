package codegen

import (
	"bytes"
	"fmt"
	a "go/ast"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"strings"

	myast "github.com/Neratus/golinq/internal/ast"
	"github.com/Neratus/golinq/internal/condition"
)

type queryData struct {
	Name             string
	Params           []param
	ResultType       string
	ModelType        string
	Method           string
	ASTLiteral       string
	ModelImportAlias string
	ModelImportPath  string
	ResultStructDef  string
	ResultStructName string
}

type param struct {
	Name string
	Type string
}

func generateResultStruct(ast *condition.SelectQueryAST, modelFields []myast.StructField) (structName, structDef string) {
	if len(ast.Joins) == 0 && len(ast.SelectFields) == len(modelFields) {
		return "", ""
	}
	mainAlias := ast.From.Alias
	hasNonMainField := false
	for _, f := range ast.SelectFields {
		if f.TableAlias != mainAlias {
			hasNonMainField = true
			break
		}
	}
	if !hasNonMainField {
		return "", ""
	}
	var hashBuilder strings.Builder
	for _, f := range ast.SelectFields {
		hashBuilder.WriteString(f.TableAlias)
		hashBuilder.WriteString(".")
		hashBuilder.WriteString(f.ColumnName)
		hashBuilder.WriteString(":")
		hashBuilder.WriteString(f.GoType)
		hashBuilder.WriteString(";")
	}
	baseName := ast.From.Alias
	joinName := ast.Joins[0].Right.Alias
	structName = "Result_" + baseName + "_" + joinName
	var defBuilder strings.Builder
	defBuilder.WriteString("type ")
	defBuilder.WriteString(structName)
	defBuilder.WriteString(" struct {\n")
	for _, f := range ast.SelectFields {
		fieldName := toCamelCase(f.ColumnName)
		defBuilder.WriteString("\t")
		defBuilder.WriteString(fieldName)
		defBuilder.WriteString(" ")
		defBuilder.WriteString(f.GoType)
		defBuilder.WriteString(" `golinq:\"column=")
		defBuilder.WriteString(f.ColumnName)
		defBuilder.WriteString("\"`")
		defBuilder.WriteString("\n")
	}
	defBuilder.WriteString("}\n")
	return structName, defBuilder.String()
}

func renderExpression(node *condition.ConditionNode) (string, error) {
	switch node.Type {
	case condition.Param:
		return fmt.Sprintf("paramValues[%d]", node.ParamIndex-1), nil
	case condition.Const:
		return renderLiteral(node.Value)
	case condition.Field:
		return "", fmt.Errorf("cannot pass model field to user function")
	case condition.Func:
		funcName, ok := node.Value.(string)
		if !ok {
			return "", fmt.Errorf("Func node value is not string: %T", node.Value)
		}
		args := make([]string, len(node.Children))
		for i, child := range node.Children {
			expr, err := renderExpression(child)
			if err != nil {
				return "", err
			}
			args[i] = expr
		}
		return funcName + "(" + strings.Join(args, ", ") + ")", nil
	default:
		return "", fmt.Errorf("unsupported node type in function argument: %v", node.Type)
	}
}

func renderConditionNode(w io.Writer, node *condition.ConditionNode, depth int) error {
	if node == nil {
		_, err := io.WriteString(w, "nil")
		return err
	}
	if node.Type == condition.Func {
		expr, err := renderExpression(node)
		if err != nil {
			return err
		}
		indent := strings.Repeat("\t", depth)
		childIndent := strings.Repeat("\t", depth+1)
		if _, err := fmt.Fprintf(w, "&condition.ConditionNode{\n"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%sType: condition.Const,\n", childIndent); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%sValue: %s,\n", childIndent, expr); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s}", indent); err != nil {
			return err
		}
		return nil
	}

	indent := strings.Repeat("\t", depth)
	childIndent := strings.Repeat("\t", depth+1)

	if _, err := fmt.Fprintf(w, "&condition.ConditionNode{\n"); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%sType: condition.%s,\n", childIndent, nodeTypeString(node.Type)); err != nil {
		return err
	}

	if needValueField(node) {
		valStr, err := renderLiteral(node.Value)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%sValue: %s,\n", childIndent, valStr); err != nil {
			return err
		}
	}

	if node.Type == condition.Param {
		if _, err := fmt.Fprintf(w, "%sParamIndex: %d,\n", childIndent, node.ParamIndex); err != nil {
			return err
		}
	}

	if len(node.Children) > 0 {
		if _, err := fmt.Fprintf(w, "%sChildren: []*condition.ConditionNode{\n", childIndent); err != nil {
			return err
		}
		for _, child := range node.Children {
			if err := renderConditionNode(w, child, depth+2); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, ",\n"); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "%s},\n", childIndent); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "%s}", indent); err != nil {
		return err
	}
	return nil
}

func renderSelectQueryAST(ast *condition.SelectQueryAST, depth int) (string, error) {
	var w bytes.Buffer
	indent := strings.Repeat("\t", depth)
	childIndent := strings.Repeat("\t", depth+1)

	if _, err := fmt.Fprintf(&w, "&condition.SelectQueryAST{\n"); err != nil {
		return "", err
	}

	if _, err := fmt.Fprintf(&w, "%sSelectFields: []condition.SelectField{\n", childIndent); err != nil {
		return "", err
	}
	for _, f := range ast.SelectFields {
		if _, err := fmt.Fprintf(&w, "%s\t{TableAlias: %q, ColumnName: %q},\n", childIndent, f.TableAlias, f.ColumnName); err != nil {
			return "", err
		}
	}
	if _, err := fmt.Fprintf(&w, "%s},\n", childIndent); err != nil {
		return "", err
	}

	if _, err := fmt.Fprintf(&w, "%sFrom: condition.Relation{Name: %q, Alias: %q},\n", childIndent, ast.From.Name, ast.From.Alias); err != nil {
		return "", err
	}

	if _, err := fmt.Fprintf(&w, "%sJoins: []condition.JoinNode{\n", childIndent); err != nil {
		return "", err
	}
	for _, j := range ast.Joins {
		if _, err := fmt.Fprintf(&w, "%s\t{\n", childIndent); err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(&w, "%s\t\tType: %q,\n", childIndent, j.Type); err != nil {
			return "", err
		}
		if j.Left == nil {
			if _, err := fmt.Fprintf(&w, "%s\t\tLeft: nil,\n", childIndent); err != nil {
				return "", err
			}
		} else {
			if _, err := fmt.Fprintf(&w, "%s\t\tLeft: &condition.Relation{Name: %q, Alias: %q},\n", childIndent, j.Left.Name, j.Left.Alias); err != nil {
				return "", err
			}
		}
		if _, err := fmt.Fprintf(&w, "%s\t\tRight: condition.Relation{Name: %q, Alias: %q},\n", childIndent, j.Right.Name, j.Right.Alias); err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(&w, "%s\t\tOn: ", childIndent); err != nil {
			return "", err
		}
		if err := renderConditionNode(&w, j.On, depth+3); err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(&w, ",\n"); err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(&w, "%s\t},\n", childIndent); err != nil {
			return "", err
		}
	}
	if _, err := fmt.Fprintf(&w, "%s},\n", childIndent); err != nil {
		return "", err
	}

	if _, err := fmt.Fprintf(&w, "%sWhere: ", childIndent); err != nil {
		return "", err
	}
	if ast.Where == nil {
		if _, err := fmt.Fprintf(&w, "nil,\n"); err != nil {
			return "", err
		}
	} else {
		if err := renderConditionNode(&w, ast.Where, depth+2); err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(&w, ",\n"); err != nil {
			return "", err
		}
	}

	if ast.OrderBy == nil {
		if _, err := fmt.Fprintf(&w, "%sOrderBy: nil,\n", childIndent); err != nil {
			return "", err
		}
	} else {
		if _, err := fmt.Fprintf(&w, "%sOrderBy: &condition.OrderByClause{\n", childIndent); err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(&w, "%s\tTableAlias: %q,\n", childIndent, ast.OrderBy.TableAlias); err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(&w, "%s\tField: %q,\n", childIndent, ast.OrderBy.Field); err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(&w, "%s\tMappingSQL: %q,\n", childIndent, ast.OrderBy.MappingSQL); err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(&w, "%s\tDesc: %t,\n", childIndent, ast.OrderBy.Desc); err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(&w, "%s},\n", childIndent); err != nil {
			return "", err
		}
	}

	if _, err := fmt.Fprintf(&w, "%sLimit: %d,\n", childIndent, ast.Limit); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(&w, "%sOffset: %d,\n", childIndent, ast.Offset); err != nil {
		return "", err
	}
	methodStr := "ToList"
	if ast.Method == condition.First {
		methodStr = "First"
	}
	if _, err := fmt.Fprintf(&w, "%sMethod: condition.%s,\n", childIndent, methodStr); err != nil {
		return "", err
	}

	if _, err := fmt.Fprintf(&w, "%s}", indent); err != nil {
		return "", err
	}
	return w.String(), nil
}

func extractParams(qspec *myast.QuerySpec) ([]param, error) {
	var params []param
	globalIdx := 0
	for _, step := range qspec.Steps {
		if step.Type != myast.StepWhere {
			continue
		}
		pred := step.PredicateRef
		for i, argExpr := range step.PredicateArgs {
			if _, ok := argExpr.(*a.BasicLit); ok {
				continue
			}
			typeName := pred.Args[i+1].TypeName
			params = append(params, param{
				Name: fmt.Sprintf("p%d", globalIdx),
				Type: typeName,
			})
			globalIdx++
		}
	}
	return params, nil
}

func Generate(queries *myast.ProjectQueries, resDir string) error {
	var buf bytes.Buffer
	var pkgName string
	for _, q := range queries.QueryCalls {
		pkgName = q.PackageName
		break
	}
	if pkgName == "" {
		for _, p := range queries.Predicates {
			pkgName = p.PackageName
			break
		}
	}
	if pkgName == "" {
		return fmt.Errorf("cannot determine package name: no queries or predicates found")
	}

	var queriesData []queryData
	for _, qspec := range queries.QueryCalls {
		ast, err := condition.BuildSelectAstTree(qspec, queries.Models, queries.FileImports)
		if err != nil {
			return fmt.Errorf("build AST for query: %w", err)
		}

		astLiteral, err := renderSelectQueryAST(ast, 0)
		if err != nil {
			return fmt.Errorf("render AST: %w", err)
		}

		params, err := extractParams(qspec)
		if err != nil {
			return fmt.Errorf("extract params: %w", err)
		}

		modelKey := qspec.ModelImportAlias + "." + qspec.StructName
		var modelFields []myast.StructField
		if m, ok := queries.Models[modelKey]; ok {
			modelFields = m.Fields
		}

		structName, structDef := generateResultStruct(ast, modelFields)

		resultType := "[]" + qspec.StructName
		modelType := qspec.StructName
		if structName != "" {
			resultType = "[]" + structName
			modelType = structName
		}
		if qspec.Method == "First" {
			resultType = strings.TrimPrefix(resultType, "[]")
		}

		hashSuffix := getHash(ast)
		queriesData = append(queriesData, queryData{
			Name:             generateFuncName(qspec, hashSuffix),
			Params:           params,
			ResultType:       resultType,
			ModelType:        modelType,
			Method:           qspec.Method,
			ASTLiteral:       astLiteral,
			ModelImportAlias: qspec.ModelImportAlias,
			ModelImportPath:  qspec.ModelImportPath,
			ResultStructDef:  structDef,
			ResultStructName: structName,
		})
	}
	modelImports := make(map[string]string)
	for _, q := range queriesData {
		if q.ModelImportPath != "" && q.ModelImportAlias != "" {
			modelImports[q.ModelImportAlias] = q.ModelImportPath
		}
	}

	structDefsMap := make(map[string]string)
	for _, qd := range queriesData {
		if qd.ResultStructDef != "" {
			structDefsMap[qd.ResultStructName] = qd.ResultStructDef
		}
	}
	structDefs := make([]string, 0, len(structDefsMap))
	for _, def := range structDefsMap {
		structDefs = append(structDefs, def)
	}
	data := struct {
		Package      string
		ModelImports map[string]string
		Queries      []queryData
		StructDefs   []string
	}{
		Package:      pkgName,
		ModelImports: modelImports,
		Queries:      queriesData,
		StructDefs:   structDefs,
	}
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	outPath := filepath.Join(resDir, "golinq_queries.gen.go")
	if err := os.MkdirAll(resDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, formatted, 0644); err != nil {
		return err
	}
	return nil
}
