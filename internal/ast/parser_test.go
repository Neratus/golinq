package ast

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestGenerateTreeOnExample(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(filename), "../../tests/example")

	proj, err := ParseDir(testdataDir)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}

	if len(proj.Models) != 2 {
		t.Errorf("expected 2 models, got %d", len(proj.Models))
	}
	if len(proj.Predicates) != 2 {
		t.Errorf("expected 2 predicates, got %d", len(proj.Predicates))
	}
	if len(proj.Joins) != 2 {
		t.Errorf("expected 2 joins, got %d", len(proj.Joins))
	}
	if len(proj.QueryCalls) != 2 {
		t.Errorf("expected 2 query calls, got %d", len(proj.QueryCalls))
	}

	var userModel *ModelMeta
	for _, model := range proj.Models {
		if model.StructName == "User" {
			userModel = model
			break
		}
	}
	if userModel == nil {
		t.Fatal("User model not found")
	}
	if userModel.TableName != "user" {
		t.Errorf("User table name = %s, want user", userModel.TableName)
	}
	expectedFields := map[string]string{
		"ID":   "user_id",
		"Name": "full_name",
		"Age":  "age",
	}
	for _, f := range userModel.Fields {
		if expected, ok := expectedFields[f.FieldName]; ok {
			if f.MappingSQL != expected {
				t.Errorf("field %s: mapping = %s, want %s", f.FieldName, f.MappingSQL, expected)
			}
		}
	}

	for _, p := range proj.Predicates {
		switch p.PredicateName {
		case "Adult":
			if p.ModelType != "models.User" && p.ModelType != "User" {
				t.Errorf("Adult.ModelType = %s, expected models.User", p.ModelType)
			}
			if p.FuncBody == nil {
				t.Error("Adult should have inline FuncBody, but got nil")
			}
			if p.LinkedFuncName != nil && *p.LinkedFuncName != "" {
				t.Error("Adult should not have LinkedFuncName")
			}
		case "ActiveUser":
			if p.LinkedFuncName == nil || *p.LinkedFuncName != "IsActive" {
				t.Errorf("ActiveUser.LinkedFuncName = %v, want IsActive", p.LinkedFuncName)
			}
		}
	}

	for _, j := range proj.Joins {
		if j.JoinName == "AnotherJoin" {
			if j.LinkedFuncName == nil || *j.LinkedFuncName != "userOrderCondition" {
				t.Errorf("AnotherJoin.LinkedFuncName = %v, want userOrderCondition", j.LinkedFuncName)
			}
			if j.LeftModelType != "models.User" && j.LeftModelType != "User" {
				t.Errorf("LeftModelType = %s, expected models.User", j.LeftModelType)
			}
		}
	}

	for _, q := range proj.QueryCalls {
		if q.Method == "ToList" && q.LimitVal == 10 && q.OffsetVal == 5 && q.OrderBy != nil && q.OrderBy.Field == "Age" {
			if q.Steps[0].Predicate != "predicates.Adult" && q.Steps[0].Predicate != "Adult" {
				t.Errorf("Query with OrderBy Age should use Adult predicate, got %s", q.Steps[0].Predicate)
			}
		}
	}

	if err := proj.Validate(); err != nil {
		t.Errorf("Validate() returned error on correct example: %v", err)
	}
}

func loadTestProject(t *testing.T, name string) *ProjectQueries {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(filename), "../../tests/example_invalid", name)
	proj, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir error for %s: %v", name, err)
	}
	return proj
}

func TestValidate_MissingModel(t *testing.T) {
	proj := loadTestProject(t, "missing_model")
	err := proj.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "argument of type User which does not match any model") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_PredicateWrongArgType(t *testing.T) {
	proj := loadTestProject(t, "predicate_wrong_arg")
	err := proj.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "argument of type") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_JoinMissingModel(t *testing.T) {
	proj := loadTestProject(t, "join_missing_model")
	err := proj.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "left argument of type Order which does not match any model") {
		t.Errorf("unexpected error message: %v", err)
	}
}
func TestValidate_QueryCallUnknownPredicate(t *testing.T) {
	proj := loadTestProject(t, "query_unknown_predicate")
	err := proj.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "struct type is not defined in any model") {
		t.Errorf("unexpected error message: %v", err)
	}
}
func TestValidate_QueryCallWrongPredicateModel(t *testing.T) {
	proj := loadTestProject(t, "query_wrong_predicate_model")
	err := proj.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "Predicate orderPredicate has argument of type Order which does not match any model") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_QueryCallUnknownJoin(t *testing.T) {
	proj := loadTestProject(t, "query_unknown_join")
	err := proj.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "struct type is not defined in any model") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_QueryCallJoinNotInvolvingModel(t *testing.T) {
	proj := loadTestProject(t, "query_join_not_involving_model")
	err := proj.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "left argument of type Order which does not match any model") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_QueryCallSelectColumnMissing(t *testing.T) {
	proj := loadTestProject(t, "query_select_column_missing")
	err := proj.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "struct type is not defined in any model") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_QueryCallOrderByMissingField(t *testing.T) {
	proj := loadTestProject(t, "query_orderby_missing_field")
	err := proj.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "struct type is not defined in any model") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) > 0 && len(s) > 0 && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || (len(s) > len(substr) && contains(s[1:], substr))))
}
