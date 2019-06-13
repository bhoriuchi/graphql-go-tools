package tools

import (
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// SchemaDirectiveVisitor defines a schema visitor.
// This attempts to provide similar functionality to Apollo graphql-tools
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
		Name:        name,
		Description: getDescription(definition),
		Args:        graphql.FieldConfigArgument{},
		Locations:   []string{},
	}

	for _, arg := range definition.Arguments {
		if argValue, err := c.buildArgFromAST(arg); err == nil {
			directiveConfig.Args[arg.Name.Value] = argValue
		} else {
			return err
		}
	}

	for _, loc := range definition.Locations {
		directiveConfig.Locations = append(directiveConfig.Locations, loc.Value)
	}

	c.directives[name] = graphql.NewDirective(directiveConfig)
	return nil
}

// applies directives
func (c *registry) applyDirectives(config interface{}, directives []*ast.Directive) error {
	if c.directiveMap == nil {
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

		switch config.(type) {
		case *graphql.SchemaConfig:
			visitor.VisitSchema(config.(*graphql.SchemaConfig), args)
		case *graphql.ScalarConfig:
			visitor.VisitScalar(config.(*graphql.ScalarConfig), args)
		case *graphql.ObjectConfig:
			visitor.VisitObject(config.(*graphql.ObjectConfig), args)
		case *graphql.Field:
			visitor.VisitFieldDefinition(config.(*graphql.Field), args)
		case *graphql.ArgumentConfig:
			visitor.VisitArgumentDefinition(config.(*graphql.ArgumentConfig), args)
		case *graphql.InterfaceConfig:
			visitor.VisitInterface(config.(*graphql.InterfaceConfig), args)
		case *graphql.UnionConfig:
			visitor.VisitUnion(config.(*graphql.UnionConfig), args)
		case *graphql.EnumConfig:
			visitor.VisitEnum(config.(*graphql.EnumConfig), args)
		case *graphql.EnumValueConfig:
			visitor.VisitEnumValue(config.(*graphql.EnumValueConfig), args)
		case *graphql.InputObjectConfig:
			visitor.VisitInputObject(config.(*graphql.InputObjectConfig), args)
		case *graphql.InputObjectFieldConfig:
			visitor.VisitInputFieldDefinition(config.(*graphql.InputObjectFieldConfig), args)
		}
	}

	return nil
}
