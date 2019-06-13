package tools

import (
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

// converts the directive map to an array
func (c *registry) directiveArray() []*graphql.Directive {
	a := make([]*graphql.Directive, 0)
	for _, d := range c.directives {
		a = append(a, d)
	}
	return a
}

// builds directives from ast
func (c *registry) buildDirectiveFromAST(definition *ast.DirectiveDefinition) error {
	name := definition.Name.Value
	directiveConfig := graphql.DirectiveConfig{
		Name:      name,
		Args:      graphql.FieldConfigArgument{},
		Locations: []string{},
	}
	if definition.Description != nil {
		directiveConfig.Description = definition.Description.Value
	}

	// add args
	for _, arg := range definition.Arguments {
		if a, err := c.buildArgFromAST(arg); err == nil {
			directiveConfig.Args[arg.Name.Value] = a
		} else {
			return err
		}
	}

	// add locations
	for _, loc := range definition.Locations {
		directiveConfig.Locations = append(directiveConfig.Locations, loc.Value)
	}

	c.directives[name] = graphql.NewDirective(directiveConfig)

	return nil
}

// applies directives
func (c *registry) applyDirectives(target interface{}, directives []*ast.Directive) error {
	if c.directiveMap == nil || directives == nil {
		return nil
	}

	directiveMap := *c.directiveMap
	for _, def := range directives {
		name := def.Name.Value
		visitor, hasVisitor := directiveMap[name]
		if !hasVisitor {
			continue
		}

		directive, err := c.getDirective(name)
		if err != nil {
			return err
		}

		args, err := getArgumentValues(directive.Args, def.Arguments, map[string]interface{}{})
		if err != nil {
			return err
		}

		switch target.(type) {
		case *graphql.SchemaConfig:
			visitor.VisitSchema(target.(*graphql.SchemaConfig), args)
		case *graphql.ScalarConfig:
			visitor.VisitScalar(target.(*graphql.ScalarConfig), args)
		case *graphql.ObjectConfig:
			visitor.VisitObject(target.(*graphql.ObjectConfig), args)
		case *graphql.Field:
			visitor.VisitFieldDefinition(target.(*graphql.Field), args)
		case *graphql.ArgumentConfig:
			visitor.VisitArgumentDefinition(target.(*graphql.ArgumentConfig), args)
		case *graphql.InterfaceConfig:
			visitor.VisitInterface(target.(*graphql.InterfaceConfig), args)
		case *graphql.UnionConfig:
			visitor.VisitUnion(target.(*graphql.UnionConfig), args)
		case *graphql.EnumConfig:
			visitor.VisitEnum(target.(*graphql.EnumConfig), args)
		case *graphql.EnumValueConfig:
			visitor.VisitEnumValue(target.(*graphql.EnumValueConfig), args)
		case *graphql.InputObjectConfig:
			visitor.VisitInputObject(target.(*graphql.InputObjectConfig), args)
		case *graphql.InputObjectFieldConfig:
			visitor.VisitInputFieldDefinition(target.(*graphql.InputObjectFieldConfig), args)
		}
	}

	return nil
}
