package tools

import (
	"fmt"
	"strings"

	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/printer"
	"github.com/graphql-go/graphql/language/source"
)

// ConcatenateTypeDefs combines one ore more typeDefs into an ast Document
func (c *ExecutableSchema) ConcatenateTypeDefs() (*ast.Document, error) {
	switch c.TypeDefs.(type) {
	case string:
		return c.concatenateTypeDefs([]string{c.TypeDefs.(string)})
	case []string:
		return c.concatenateTypeDefs(c.TypeDefs.([]string))
	case func() []string:
		return c.concatenateTypeDefs(c.TypeDefs.(func() []string)())
	}
	return nil, fmt.Errorf("Unsupported TypeDefs value. Must be one of string, []string, or func() []string")
}

// performs the actual concatenation of the types by parsing each
// typeDefs string and converting each definition into a string
// then creating a unique list of all definitions and finally
// printing them as a single definition and returning the parsed document
func (c *ExecutableSchema) concatenateTypeDefs(typeDefs []string) (*ast.Document, error) {
	resolvedTypes := map[string]interface{}{}
	for _, defs := range typeDefs {
		doc, err := parser.Parse(parser.ParseParams{
			Source: &source.Source{
				Body: []byte(defs),
				Name: "GraphQL",
			},
		})
		if err != nil {
			return nil, err
		}

		for _, typeDef := range doc.Definitions {
			if def := printer.Print(typeDef); def != nil {
				stringDef := strings.TrimSpace(def.(string))
				resolvedTypes[stringDef] = nil
			}
		}
	}

	typeArray := []string{}
	for def := range resolvedTypes {
		typeArray = append(typeArray, def)
	}

	return parser.Parse(parser.ParseParams{
		Source: &source.Source{
			Body: []byte(strings.Join(typeArray, "\n")),
			Name: "GraphQL",
		},
	})
}
