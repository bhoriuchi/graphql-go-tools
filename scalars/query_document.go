package scalars

import (
	"encoding/json"
	"regexp"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
)

var queryDocOperatorRx = regexp.MustCompile(`^\$`)
var storedQueryDocOperatorRx = regexp.MustCompile(`^_`)

func replacePrefixedKeys(obj interface{}, prefixRx *regexp.Regexp, replacement string) interface{} {
	switch obj.(type) {
	case map[string]interface{}:
		result := map[string]interface{}{}
		for k, v := range obj.(map[string]interface{}) {
			newKey := prefixRx.ReplaceAllString(k, replacement)
			result[newKey] = replacePrefixedKeys(v, prefixRx, replacement)
		}
		return result

	case []interface{}:
		result := []interface{}{}
		for _, v := range obj.([]interface{}) {
			result = append(result, replacePrefixedKeys(v, prefixRx, replacement))
		}
		return result

	default:
		return obj
	}
}

func serializeQueryDocFn(value interface{}) interface{} {
	return replacePrefixedKeys(value, storedQueryDocOperatorRx, "$")
}

func parseValueQueryDocFn(value interface{}) interface{} {
	return replacePrefixedKeys(value, queryDocOperatorRx, "_")
}

func parseLiteralQueryDocFn(astValue ast.Value) interface{} {
	var val interface{}
	switch astValue.GetKind() {
	case kinds.StringValue:
		bvalue := []byte(astValue.GetValue().(string))
		if err := json.Unmarshal(bvalue, &val); err != nil {
			return nil
		}
		return replacePrefixedKeys(val, queryDocOperatorRx, "_")
	case kinds.ObjectValue:
		return parseLiteralJSONFn(astValue)
	}
	return nil
}

// ScalarQueryDocument a mongodb style query document
var ScalarQueryDocument = graphql.NewScalar(
	graphql.ScalarConfig{
		Name:         "QueryDocument",
		Description:  "MongoDB style query document",
		Serialize:    serializeQueryDocFn,
		ParseValue:   parseValueQueryDocFn,
		ParseLiteral: parseLiteralQueryDocFn,
	},
)
