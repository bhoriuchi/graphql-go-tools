package tools

// taken from https://github.com/graphql-go/graphql/values.go
// since none of these functions are exported

import (
	"math"
	"reflect"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
)

// Prepares an object map of argument values given a list of argument
// definitions and list of argument AST nodes.
func getArgumentValues(argDefs []*graphql.Argument, argASTs []*ast.Argument, variableVariables map[string]interface{}) (map[string]interface{}, error) {

	argASTMap := map[string]*ast.Argument{}
	for _, argAST := range argASTs {
		if argAST.Name != nil {
			argASTMap[argAST.Name.Value] = argAST
		}
	}
	results := map[string]interface{}{}
	for _, argDef := range argDefs {

		name := argDef.PrivateName
		var valueAST ast.Value
		if argAST, ok := argASTMap[name]; ok {
			valueAST = argAST.Value
		}
		value := valueFromAST(valueAST, argDef.Type, variableVariables)
		if isNullish(value) {
			value = argDef.DefaultValue
		}
		if !isNullish(value) {
			results[name] = value
		}
	}
	return results, nil
}

// Returns true if a value is null, undefined, or NaN.
func isNullish(src interface{}) bool {
	if src == nil {
		return true
	}
	value := reflect.ValueOf(src)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.String:
		// if src is ptr type and len(string)=0, it returns false
		if !value.IsValid() {
			return true
		}
	case reflect.Int:
		return math.IsNaN(float64(value.Int()))
	case reflect.Float32, reflect.Float64:
		return math.IsNaN(float64(value.Float()))
	}
	return false
}

/**
 * Produces a value given a GraphQL Value AST.
 *
 * A GraphQL type must be provided, which will be used to interpret different
 * GraphQL Value literals.
 *
 * | GraphQL Value        | JSON Value    |
 * | -------------------- | ------------- |
 * | Input Object         | Object        |
 * | List                 | Array         |
 * | Boolean              | Boolean       |
 * | String / Enum Value  | String        |
 * | Int / Float          | Number        |
 *
 */
func valueFromAST(valueAST ast.Value, ttype graphql.Input, variables map[string]interface{}) interface{} {

	if ttype, ok := ttype.(*graphql.NonNull); ok {
		val := valueFromAST(valueAST, ttype.OfType, variables)
		return val
	}

	if valueAST == nil {
		return nil
	}

	if valueAST, ok := valueAST.(*ast.Variable); ok && valueAST.Kind == kinds.Variable {
		if valueAST.Name == nil {
			return nil
		}
		if variables == nil {
			return nil
		}
		variableName := valueAST.Name.Value
		variableVal, ok := variables[variableName]
		if !ok {
			return nil
		}
		// Note: we're not doing any checking that this variable is correct. We're
		// assuming that this query has been validated and the variable usage here
		// is of the correct type.
		return variableVal
	}

	if ttype, ok := ttype.(*graphql.List); ok {
		itemType := ttype.OfType
		if valueAST, ok := valueAST.(*ast.ListValue); ok && valueAST.Kind == kinds.ListValue {
			values := []interface{}{}
			for _, itemAST := range valueAST.Values {
				v := valueFromAST(itemAST, itemType, variables)
				values = append(values, v)
			}
			return values
		}
		v := valueFromAST(valueAST, itemType, variables)
		return []interface{}{v}
	}

	if ttype, ok := ttype.(*graphql.InputObject); ok {
		valueAST, ok := valueAST.(*ast.ObjectValue)
		if !ok {
			return nil
		}
		fieldASTs := map[string]*ast.ObjectField{}
		for _, fieldAST := range valueAST.Fields {
			if fieldAST.Name == nil {
				continue
			}
			fieldName := fieldAST.Name.Value
			fieldASTs[fieldName] = fieldAST

		}
		obj := map[string]interface{}{}
		for fieldName, field := range ttype.Fields() {
			fieldAST, ok := fieldASTs[fieldName]
			fieldValue := field.DefaultValue
			if !ok || fieldAST == nil {
				if fieldValue == nil {
					continue
				}
			} else {
				fieldValue = valueFromAST(fieldAST.Value, field.Type, variables)
			}
			if isNullish(fieldValue) {
				fieldValue = field.DefaultValue
			}
			if !isNullish(fieldValue) {
				obj[fieldName] = fieldValue
			}
		}
		return obj
	}

	switch ttype := ttype.(type) {
	case *graphql.Scalar:
		parsed := ttype.ParseLiteral(valueAST)
		if !isNullish(parsed) {
			return parsed
		}
	case *graphql.Enum:
		parsed := ttype.ParseLiteral(valueAST)
		if !isNullish(parsed) {
			return parsed
		}
	}
	return nil
}
