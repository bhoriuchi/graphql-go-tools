package tools

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/kinds"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
)

// MakeExecutableSchemaConfig configuration for making an executable schema
// this attempts to provide similar functionality to Apollo graphql-tools
// https://www.apollographql.com/docs/graphql-tools/generate-schema
type MakeExecutableSchemaConfig struct {
	TypeDefs         string
	Types            *map[string]graphql.Type
	Resolvers        *ResolverMap
	SchemaDirectives *SchemaDirectiveVisitorMap
	Directives       *DirectiveMap
}

// MakeExecutableSchema creates an executable graphql schema
func MakeExecutableSchema(config MakeExecutableSchemaConfig) (graphql.Schema, error) {
	// create a registry with the resolver map
	registry := newTypeRegistry(config.Resolvers, config.SchemaDirectives)

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
	astDocument, err := parser.Parse(parser.ParseParams{
		Source: &source.Source{
			Body: []byte(config.TypeDefs),
			Name: "GraphQL",
		},
	})
	if err != nil {
		return graphql.Schema{}, err
	}

	// build types in order of possible dependencies
	if err := registry.buildTypesFromASTDocument(astDocument, kinds.DirectiveDefinition); err != nil {
		return graphql.Schema{}, err
	} else if err := registry.buildTypesFromASTDocument(astDocument, kinds.ScalarDefinition); err != nil {
		return graphql.Schema{}, err
	} else if err := registry.buildTypesFromASTDocument(astDocument, kinds.EnumDefinition); err != nil {
		return graphql.Schema{}, err
	} else if err := registry.buildTypesFromASTDocument(astDocument, kinds.InputObjectDefinition); err != nil {
		return graphql.Schema{}, err
	} else if err := registry.buildTypesFromASTDocument(astDocument, kinds.ObjectDefinition); err != nil {
		return graphql.Schema{}, err
	} else if err := registry.buildTypesFromASTDocument(astDocument, kinds.InterfaceDefinition); err != nil {
		return graphql.Schema{}, err
	} else if err := registry.buildTypesFromASTDocument(astDocument, kinds.UnionDefinition); err != nil {
		return graphql.Schema{}, err
	} else if err := registry.buildTypesFromASTDocument(astDocument, kinds.SchemaDefinition); err != nil {
		return graphql.Schema{}, err
	}

	// look for an object type with the name Query
	rootQueryType, err := registry.getObject(registry.rootQueryName)
	if err != nil {
		return graphql.Schema{}, fmt.Errorf("root query object with name %q not found", registry.rootQueryName)
	}

	// get other root resolvers types
	mutation, err := registry.getObject(registry.rootMutationName)
	if err != nil && registry.definedMutationName {
		return graphql.Schema{}, fmt.Errorf("root mutation object with name %q not found", registry.rootMutationName)
	}

	subscription, err := registry.getObject(registry.rootSubscriptionName)
	if err != nil && registry.definedSubscriptionName {
		return graphql.Schema{}, fmt.Errorf("root subscription object with name %q not found", registry.rootSubscriptionName)
	}

	// create a new schema config
	schemaConfig := graphql.SchemaConfig{
		Query:        rootQueryType,
		Mutation:     mutation,
		Subscription: subscription,
		Types:        registry.typeArray(),
		Directives:   registry.directiveArray(),
	}

	// apply schema directives
	if err := registry.applyDirectives(&schemaConfig, registry.schemaDirectives); err != nil {
		return graphql.Schema{}, err
	}

	// create a new schema
	return graphql.NewSchema(schemaConfig)
}
