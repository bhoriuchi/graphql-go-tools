package tools

import (
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
)

// default root type names
const (
	DefaultRootQueryName        = "Query"
	DefaultRootMutationName     = "Mutation"
	DefaultRootSubscriptionName = "Subscription"
)

// MakeExecutableSchema is shorthand for ExecutableSchema{}.Make()
func MakeExecutableSchema(config ExecutableSchema) (graphql.Schema, error) {
	return config.Make()
}

// ExecutableSchema configuration for making an executable schema
// this attempts to provide similar functionality to Apollo graphql-tools
// https://www.apollographql.com/docs/graphql-tools/generate-schema
type ExecutableSchema struct {
	TypeDefs         interface{}
	Types            *map[string]graphql.Type
	Resolvers        *ResolverMap
	SchemaDirectives *SchemaDirectiveVisitorMap
	Directives       *DirectiveMap
}

// Make creates an executable graphql schema
func (c *ExecutableSchema) Make() (graphql.Schema, error) {
	// combine the TypeDefs
	document, err := c.ConcatenateTypeDefs()
	if err != nil {
		return graphql.Schema{}, err
	}

	// create a new registry
	registry := newRegistry(c.Resolvers, c.SchemaDirectives, document)

	// add additional types to the registry
	if c.Types != nil {
		for name, t := range *c.Types {
			registry.setType(name, t)
		}
	}

	// add additional directives to the registry
	if c.Directives != nil {
		for name, d := range *c.Directives {
			registry.setDirective(name, d)
		}
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
	query, _ := registry.getObject(DefaultRootQueryName)
	mutation, _ := registry.getObject(DefaultRootMutationName)
	subscription, _ := registry.getObject(DefaultRootSubscriptionName)

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

// build a schema from an ast
func (c *registry) buildSchemaFromAST(definition *ast.SchemaDefinition) error {
	schemaConfig := graphql.SchemaConfig{
		Types:      c.typeArray(),
		Directives: c.directiveArray(),
	}

	// add operations
	for _, op := range definition.OperationTypes {
		switch op.Operation {
		case ast.OperationTypeQuery:
			if object, err := c.getObject(op.Type.Name.Value); err == nil {
				schemaConfig.Query = object
			} else {
				return err
			}
		case ast.OperationTypeMutation:
			if object, err := c.getObject(op.Type.Name.Value); err == nil {
				schemaConfig.Mutation = object
			} else {
				return err
			}
		case ast.OperationTypeSubscription:
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
	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		return err
	}

	c.schema = &schema
	return nil
}
