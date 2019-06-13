package tools

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// SchemaDirectiveVisitor defines a schema visitor
// this attempts to provide similar functionality to Apollo graphql-tools
// https://www.apollographql.com/docs/graphql-tools/schema-directives/
type SchemaDirectiveVisitor struct {
	VisitSchema               func(schema *graphql.SchemaConfig, args map[string]interface{})
	VisitScalar               func(scalar *graphql.ScalarConfig, args map[string]interface{})
	VisitObject               func(object *graphql.ObjectConfig, args map[string]interface{})
	VisitFieldDefinition      func(field *graphql.Field, args map[string]interface{})
	VisitArgumentDefinition   func(argument *graphql.ArgumentConfig, args map[string]interface{})
	VisitInterface            func(iface *graphql.InterfaceConfig, args map[string]interface{})
	VisitUnion                func(union *graphql.UnionConfig, args map[string]interface{})
	VisitEnum                 func(enum *graphql.EnumConfig, args map[string]interface{})
	VisitEnumValue            func(value *graphql.EnumValueConfig, args map[string]interface{})
	VisitInputObject          func(object *graphql.InputObjectConfig, args map[string]interface{})
	VisitInputFieldDefinition func(field *graphql.InputObjectFieldConfig, args map[string]interface{})
}

// SchemaDirectiveVisitorMap a map of schema directive visitors
type SchemaDirectiveVisitorMap map[string]SchemaDirectiveVisitor

// DirectiveMap a map of directives
type DirectiveMap map[string]*graphql.Directive

// Get gets a directive from the registry
func (c *typeRegistry) getDirective(name string) (*graphql.Directive, error) {
	if val, ok := c.directives[name]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("directive %q not found", name)
}

// Set sets a graphql directive in the registry
func (c *typeRegistry) setDirective(name string, graphqlDirective *graphql.Directive) {
	c.directives[name] = graphqlDirective
}

// converts the directive map to an array
func (c *typeRegistry) directiveArray() []*graphql.Directive {
	a := make([]*graphql.Directive, 0)
	for _, d := range c.directives {
		a = append(a, d)
	}
	return a
}

// builds directives from ast
func (c *typeRegistry) buildDirectiveFromAST(definition *ast.DirectiveDefinition) error {
	name := definition.Name.Value

	// build args

	directiveConfig := graphql.DirectiveConfig{
		Name:      name,
		Args:      graphql.FieldConfigArgument{},
		Locations: []string{},
	}

	// add args
	for _, arg := range definition.Arguments {
		if arg != nil {
			a, err := c.buildArgFromAST(arg)
			if err != nil {
				return err
			}
			directiveConfig.Args[arg.Name.Value] = a
		}
	}

	// add locations
	for _, loc := range definition.Locations {
		directiveConfig.Locations = append(directiveConfig.Locations, loc.Value)
	}

	if definition.Description != nil {
		directiveConfig.Description = definition.Description.Value
	}

	c.directives[name] = graphql.NewDirective(directiveConfig)

	return nil
}
