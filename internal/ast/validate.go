package ast

import (
	"fmt"
	"go/ast"
)

func (queries *ProjectQueries) Validate() error {
	for _, predicate := range queries.Predicates {
		if predicate.LinkedFuncName != nil && *predicate.LinkedFuncName != "" && len(predicate.Args) == 0 {
			key := predicate.PackageName + "." + *predicate.LinkedFuncName
			if fdecl, ok := queries.FuncDecls[key]; ok && fdecl.FuncDecl.Type.Params != nil {
				for _, param := range fdecl.FuncDecl.Type.Params.List {
					for _, name := range param.Names {
						predicate.Args = append(predicate.Args, &PredicateArg{
							Name:        name.Name,
							TypeName:    typeToString(param.Type),
							PackageName: predicate.PackageName,
							Declared:    false,
						})
					}
				}
			}
		}
		for _, arg := range predicate.Args {
			if _, ok := queries.Models[arg.TypeName]; !ok {
				return &ValidationError{Message: "Predicate " + predicate.PredicateName + " has argument of type " + arg.TypeName + " which does not match any model"}
			}
			arg.Declared = true
		}
		hasBody := predicate.FuncBody != nil
		hasLink := predicate.LinkedFuncName != nil && *predicate.LinkedFuncName != ""
		if hasBody && hasLink {
			return &ValidationError{Message: "Predicate " + predicate.PredicateName + " has both a function body and a linked function, only one is allowed"}
		} else if hasBody {
			continue
		} else if hasLink {
			key := predicate.PackageName + "." + *predicate.LinkedFuncName
			fdecl, ok := queries.FuncDecls[key]
			if !ok {
				return &ValidationError{Message: "Predicate " + predicate.PredicateName + " is linked to function " + key + " which is not defined"}
			}
			predicate.FuncDeclRef = fdecl.FuncDecl
			fdecl.IsUsed = true
		} else {
			return &ValidationError{Message: "Predicate " + predicate.PredicateName + " has neither a function body nor a linked function, one is required"}
		}

	}
	for _, join := range queries.Joins {
		if _, ok := queries.Models[join.LeftModelType]; !ok {
			return &ValidationError{Message: "Join " + join.JoinName + " has left argument of type " + join.LeftModelType + " which does not match any model"}
		}
		if _, ok := queries.Models[join.RightModelType]; !ok {
			return &ValidationError{Message: "Join " + join.JoinName + " has right argument of type " + join.RightModelType + " which does not match any model"}
		}
		if join.OnFunc != nil {
			if join.LeftArg.TypeName != "" {
				if _, ok := queries.Models[join.LeftArg.TypeName]; !ok {
					return &ValidationError{Message: "Join " + join.JoinName + " has argument of type " + join.LeftArg.TypeName + " which does not match any model"}
				}
				join.LeftArg.Declared = true
			}
			if join.RightArg.TypeName != "" {
				if _, ok := queries.Models[join.RightArg.TypeName]; !ok {
					return &ValidationError{Message: "Join " + join.JoinName + " has argument of type " + join.RightArg.TypeName + " which does not match any model"}
				}
				join.RightArg.Declared = true
			}
		}

		hasBody := join.OnFunc != nil
		hasLink := join.LinkedFuncName != nil && *join.LinkedFuncName != ""
		if hasBody && hasLink {
			return &ValidationError{Message: "Join " + join.JoinName + " has both an ON function and a linked function, only one is allowed"}
		} else if hasBody {
			continue
		} else if hasLink {
			key := join.PackageName + "." + *join.LinkedFuncName
			fdecl, ok := queries.FuncDecls[key]
			if !ok {
				return &ValidationError{Message: "Join " + join.JoinName + " is linked to function " + key + " which is not defined"}
			}
			join.FuncDeclRef = fdecl.FuncDecl
			fdecl.IsUsed = true
		} else {
			return &ValidationError{Message: "Join " + join.JoinName + " has neither an ON function nor a linked function , one of these is required"}
		}
	}
	for _, call := range queries.QueryCalls {
		fullModelName := call.StructName
		model, ok := queries.Models[fullModelName]
		if !ok {
			return &ValidationError{Message: "Query call on struct " + call.StructName + " but struct type is not defined in any model"}
		}
		for _, selectCall := range call.SelectCols {
			found := false
			for _, field := range model.Fields {
				if field.FieldName == selectCall {
					found = true
					break
				}
			}
			if !found {
				return &ValidationError{Message: "Query call on struct " + call.StructName + " has select column " + selectCall + " which does not match any field mapping in the model"}
			}
		}
		currentModel := call.StructName
		for i, step := range call.Steps {

			switch step.Type {
			case StepWhere:
				pred, ok := queries.Predicates[step.Predicate]
				if !ok {
					return &ValidationError{Message: "Query call on struct " + call.StructName + " uses predicate " + step.Predicate + " which is not defined"}
				}
				call.Steps[i].PredicateRef = pred
				expectedCount := len(pred.Args) - 1
				if len(step.PredicateArgs) != expectedCount {
					return fmt.Errorf("predicate %s expects %d argument(s), got %d", pred.PredicateName, expectedCount, len(step.PredicateArgs))
				}

				for i, argExpr := range step.PredicateArgs {
					expectedType := pred.Args[i+1].TypeName

					switch expr := argExpr.(type) {
					case *ast.BasicLit:
						if !isBasicLitCompatible(expr, expectedType) {
							return fmt.Errorf("argument %d of predicate %s: expected %s, but got literal %s",
								i+1, pred.PredicateName, expectedType, expr.Value)
						}
					case *ast.Ident:
					default:
						return fmt.Errorf("argument %d of predicate %s must be literal or variable, got %T",
							i+1, pred.PredicateName, expr)
					}
				}
				predModelShort := trimPackage(pred.ModelType)
				currentShort := trimPackage(currentModel)
				if predModelShort != currentShort {
					return &ValidationError{Message: fmt.Sprintf("predicate %s expects model %s, but current model is %s",
						pred.PredicateName, pred.ModelType, currentModel)}
				}
			case StepJoin:
				join, ok := queries.Joins[step.Join]
				if !ok {
					return &ValidationError{Message: "Query call on struct " + call.StructName + " uses join " + step.Join + " which is not defined"}
				}
				call.Steps[i].JoinRef = join

				leftShort := trimPackage(join.LeftModelType)
				rightShort := trimPackage(join.RightModelType)
				currentShort := trimPackage(currentModel)

				if leftShort == currentShort {
					currentModel = rightShort
				} else if rightShort == currentShort {
					currentModel = leftShort
				} else {
					return &ValidationError{Message: fmt.Sprintf("join %s does not involve current model %s", join.JoinName, currentModel)}
				}
			}
		}
		if call.OrderBy != nil {
			found := false
			for _, f := range model.Fields {
				if f.FieldName == call.OrderBy.Field {
					found = true
					break
				}
			}
			if !found {
				return &ValidationError{Message: "Query call on struct " + call.StructName + " has OrderBy field " + call.OrderBy.Field + " which does not match any field in the model"}
			}
		}
	}
	toDelete := []string{}
	for key, fun := range queries.FuncDecls {
		if !fun.IsUsed {
			toDelete = append(toDelete, key)
		}
	}
	for _, key := range toDelete {
		delete(queries.FuncDecls, key)
	}
	return nil
}
