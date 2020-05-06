package tools

import (
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
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

// MakeSchemaConfig creates a schema config that maintain intact types
func MakeSchemaConfig(config ExecutableSchema) (graphql.SchemaConfig, error) {
	return config.MakeSchemaConfig()
}

// ExecutableSchema configuration for making an executable schema
// this attempts to provide similar functionality to Apollo graphql-tools
// https://www.apollographql.com/docs/graphql-tools/generate-schema
type ExecutableSchema struct {
	TypeDefs         interface{}               // a string, []string, or func() []string
	Resolvers        map[string]Resolver       // a map of Resolver, Directive, Scalar, Enum, Object, InputObject, Union, or Interface
	SchemaDirectives SchemaDirectiveVisitorMap // Map of SchemaDirectiveVisitor
	Extensions       []graphql.Extension       // GraphQL extensions
}

// MakeSchemaConfig creates a graphql schema config, this struct maintains intact the types and does not require the use of a non empty Query
func (c *ExecutableSchema) MakeSchemaConfig() (graphql.SchemaConfig, error) {
	// combine the TypeDefs
	document, err := c.ConcatenateTypeDefs()
	if err != nil {
		return graphql.SchemaConfig{}, err
	}

	// create a new registry
	registry := newRegistry(c.Resolvers, c.SchemaDirectives, c.Extensions, document)

	// resolve the document definitions
	if err := registry.resolveDefinitions(); err != nil {
		return graphql.SchemaConfig{}, err
	}

	// check if schema was created by definition
	if registry.schemaConfig != nil {
		return *registry.schemaConfig, nil
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
		Extensions:   c.Extensions,
	}
	return schemaConfig, nil
}

// Make creates an executable graphql schema
func (c *ExecutableSchema) Make() (graphql.Schema, error) {
	schemaConfig, err := c.MakeSchemaConfig()
	if err != nil {
		return graphql.Schema{}, err
	}
	// create a new schema
	return graphql.NewSchema(schemaConfig)
}

// build a schema from an ast
func (c *registry) buildSchemaFromAST(definition *ast.SchemaDefinition, allowThunks bool) error {
	schemaConfig := graphql.SchemaConfig{
		Types:      c.typeArray(),
		Directives: c.directiveArray(),
		Extensions: c.extensions,
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
	if err := c.applyDirectives(&schemaConfig, definition.Directives, allowThunks); err != nil {
		return err
	}

	// build the schema
	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		return err
	}

	c.schema = &schema
	c.schemaConfig = &schemaConfig
	return nil
}
