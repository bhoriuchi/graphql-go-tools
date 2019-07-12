package scalars

import (
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
)

// ScalarJSON a scalar JSON type
var ScalarJSON = graphql.NewScalar(
	graphql.ScalarConfig{
		Name:        "JSON",
		Description: "The `JSON` scalar type represents JSON values as specified by [ECMA-404](http://www.ecma-international.org/publications/files/ECMA-ST/ECMA-404.pdf)",
		Serialize: func(value interface{}) interface{} {
			return value
		},
		ParseValue: func(value interface{}) interface{} {
			return value
		},
		ParseLiteral: parseLiteralJSONFn,
	},
)

// recursively parse ast
func parseLiteralJSONFn(astValue ast.Value) interface{} {
	switch kind := astValue.GetKind(); kind {
	// get value for primitive types
	case kinds.StringValue, kinds.BooleanValue, kinds.IntValue, kinds.FloatValue:
		return astValue.GetValue()

	// make a map for objects
	case kinds.ObjectValue:
		obj := make(map[string]interface{})
		for _, v := range astValue.GetValue().([]*ast.ObjectField) {
			obj[v.Name.Value] = parseLiteralJSONFn(v.Value)
		}
		return obj

	// make a slice for lists
	case kinds.ListValue:
		list := make([]interface{}, 0)
		for _, v := range astValue.GetValue().([]ast.Value) {
			list = append(list, parseLiteralJSONFn(v))
		}
		return list

	// default to nil
	default:
		return nil
	}
}
