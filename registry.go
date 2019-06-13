package tools

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// registry the registry holds all of the types
type registry struct {
	types                   map[string]graphql.Type
	directives              map[string]*graphql.Directive
	schema                  *graphql.Schema
	resolverMap             *ResolverMap
	directiveMap            *SchemaDirectiveVisitorMap
	rootQueryName           string
	rootMutationName        string
	definedMutationName     bool
	definedSubscriptionName bool
	rootSubscriptionName    string
	schemaDirectives        []*ast.Directive
}

// newRegistry creates a new registry
func newRegistry(resolvers *ResolverMap, directives *SchemaDirectiveVisitorMap) *registry {
	return &registry{
		types: map[string]graphql.Type{
			"ID":      graphql.ID,
			"String":  graphql.String,
			"Int":     graphql.Int,
			"Float":   graphql.Float,
			"Boolean": graphql.Boolean,
		},
		directives: map[string]*graphql.Directive{
			"include":    graphql.IncludeDirective,
			"skip":       graphql.SkipDirective,
			"deprecated": graphql.DeprecatedDirective,
		},
		resolverMap:          resolvers,
		directiveMap:         directives,
		rootQueryName:        "Query",
		rootMutationName:     "Mutation",
		rootSubscriptionName: "Subscription",
		schemaDirectives:     []*ast.Directive{},
	}
}

func (c *registry) getResolver(name string) Resolver {
	if c.resolverMap != nil {
		resolverMap := *c.resolverMap
		if resolver, ok := resolverMap[name]; ok {
			return resolver
		}
	}
	return nil
}

// gets an object from the registry
func (c *registry) getObject(name string) (*graphql.Object, error) {
	obj, err := c.getType(name)
	if err != nil {
		return nil, err
	}
	switch obj.(type) {
	case *graphql.Object:
		return obj.(*graphql.Object), nil
	}
	return nil, nil
}

// converts the type map to an array
func (c *registry) typeArray() []graphql.Type {
	a := make([]graphql.Type, 0)
	for _, t := range c.types {
		a = append(a, t)
	}
	return a
}

// Get gets a type from the registry
func (c *registry) getType(name string) (graphql.Type, error) {
	if val, ok := c.types[name]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("type %q not found", name)
}

// Set sets a graphql type in the registry
func (c *registry) setType(name string, graphqlType graphql.Type) {
	c.types[name] = graphqlType
}

// Get gets a directive from the registry
func (c *registry) getDirective(name string) (*graphql.Directive, error) {
	if val, ok := c.directives[name]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("directive %q not found", name)
}

// Set sets a graphql directive in the registry
func (c *registry) setDirective(name string, graphqlDirective *graphql.Directive) {
	c.directives[name] = graphqlDirective
}
