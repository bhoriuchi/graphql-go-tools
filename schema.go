package tools

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
)

// MakeExecutableSchemaConfig configuration for making an executable schema
// this attempts to provide similar functionality to Apollo graphql-tools
// https://www.apollographql.com/docs/graphql-tools/generate-schema
type MakeExecutableSchemaConfig struct {
	TypeDefs         interface{}
	Types            *map[string]graphql.Type
	Resolvers        *ResolverMap
	SchemaDirectives *SchemaDirectiveVisitorMap
	Directives       *DirectiveMap
}

// MakeExecutableSchema creates an executable graphql schema
func MakeExecutableSchema(config MakeExecutableSchemaConfig) (graphql.Schema, error) {
	registry := newRegistry(config.Resolvers, config.SchemaDirectives)

	// add additional types to the registry
	if config.Types != nil {
		for name, t := range *config.Types {
			registry.setType(name, t)
		}
	}

	// add additional directives to the registry
	if config.Directives != nil {
		for name, d := range *config.Directives {
			registry.setDirective(name, d)
		}
	}

	// parse the TypeDefs
	document, err := parseTypeDefs(config.TypeDefs)
	if err != nil {
		return graphql.Schema{}, err
	}

	// build types in order of possible dependencies
	buildKinds := []string{
		kinds.DirectiveDefinition,
		kinds.ScalarDefinition,
		kinds.EnumDefinition,
		kinds.InputObjectDefinition,
		kinds.ObjectDefinition,
		kinds.InterfaceDefinition,
		kinds.UnionDefinition,
		kinds.SchemaDefinition,
	}

	for _, kind := range buildKinds {
		if err := registry.buildTypeFromDocument(document, kind); err != nil {
			return graphql.Schema{}, err
		}
	}

	// check if schema was created by definition
	if registry.schema != nil {
		return *registry.schema, nil
	}

	// otherwise build a schema from default object names
	query, _ := registry.getObject("Query")
	mutation, _ := registry.getObject("Mutation")
	subscription, _ := registry.getObject("Subscription")

	// create a new schema config
	schemaConfig := graphql.SchemaConfig{
		Query:        query,
		Mutation:     mutation,
		Subscription: subscription,
		Types:        registry.typeArray(),
		Directives:   registry.directiveArray(),
	}

	// create a new schema
	return graphql.NewSchema(schemaConfig)
}

// parses the typedefs into an ast document
func parseTypeDefs(typeDefs interface{}) (*ast.Document, error) {
	switch typeDefs.(type) {
	case string:
		return parser.Parse(parser.ParseParams{
			Source: &source.Source{
				Body: []byte(typeDefs.(string)),
				Name: "GraphQL",
			},
		})
	}
	return nil, fmt.Errorf("unsupported TypeDefs value")
}

// build a schema from an ast
func (c *registry) buildSchemaFromAST(definition *ast.SchemaDefinition) error {
	schemaConfig := graphql.SchemaConfig{
		Types:      c.typeArray(),
		Directives: c.directiveArray(),
	}

	// add operations
	for _, op := range definition.OperationTypes {
		switch op.Operation {
		case "query":
			if object, err := c.getObject(op.Type.Name.Value); err == nil {
				schemaConfig.Query = object
			} else {
				return err
			}
		case "mutation":
			if object, err := c.getObject(op.Type.Name.Value); err == nil {
				schemaConfig.Mutation = object
			} else {
				return err
			}
		case "subscription":
			if object, err := c.getObject(op.Type.Name.Value); err == nil {
				schemaConfig.Subscription = object
			} else {
				return err
			}
		}
	}

	// apply schema directives
	if err := c.applyDirectives(&schemaConfig, definition.Directives); err != nil {
		return err
	}

	// build the schema
	if schema, err := graphql.NewSchema(schemaConfig); err == nil {
		c.schema = &schema
	} else {
		return err
	}
	return nil
}
