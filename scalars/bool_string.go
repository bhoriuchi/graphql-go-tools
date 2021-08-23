package scalars

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// ScalarBoolString converts boolean to a string
var ScalarBoolString = graphql.NewScalar(
	graphql.ScalarConfig{
		Name:        "BoolString",
		Description: "BoolString converts a boolean to/from a string",
		Serialize: func(value interface{}) interface{} {
			valStr := fmt.Sprintf("%v", value)
			return valStr == "true" || valStr == "1"
		},
		ParseValue: func(value interface{}) interface{} {
			b, ok := value.(bool)
			if !ok {
				return "false"
			} else if b {
				return "true"
			}
			return "false"
		},
		ParseLiteral: func(astValue ast.Value) interface{} {
			value := astValue.GetValue()
			b, ok := value.(bool)
			if !ok {
				return "false"
			} else if b {
				return "true"
			}
			return "false"
		},
	},
)
