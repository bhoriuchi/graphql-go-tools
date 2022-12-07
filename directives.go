package tools

import (
	"context"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

const (
	directiveHide = "hide"
)

// HideDirective hides a define field
var HideDirective = graphql.NewDirective(graphql.DirectiveConfig{
	Name:        directiveHide,
	Description: "Hide a field, useful when generating types from the AST where the backend type has more fields than the graphql type",
	Locations:   []string{graphql.DirectiveLocationFieldDefinition},
	Args:        graphql.FieldConfigArgument{},
})

// SchemaDirectiveVisitor defines a schema visitor.
// This attempts to provide similar functionality to Apollo graphql-tools
// https://www.apollographql.com/docs/graphql-tools/schema-directives/
type SchemaDirectiveVisitor struct {
	VisitSchema               func(p VisitSchemaParams) error
	VisitScalar               func(p VisitScalarParams) error
	VisitObject               func(p VisitObjectParams) error
	VisitFieldDefinition      func(p VisitFieldDefinitionParams) error
	VisitArgumentDefinition   func(p VisitArgumentDefinitionParams) error
	VisitInterface            func(p VisitInterfaceParams) error
	VisitUnion                func(p VisitUnionParams) error
	VisitEnum                 func(p VisitEnumParams) error
	VisitEnumValue            func(p VisitEnumValueParams) error
	VisitInputObject          func(p VisitInputObjectParams) error
	VisitInputFieldDefinition func(p VisitInputFieldDefinitionParams) error
}

// VisitSchemaParams params
type VisitSchemaParams struct {
	Context context.Context
	Config  *graphql.SchemaConfig
	Node    *ast.SchemaDefinition
	Args    map[string]interface{}
}

// VisitScalarParams params
type VisitScalarParams struct {
	Context context.Context
	Config  *graphql.ScalarConfig
	Node    *ast.ScalarDefinition
	Args    map[string]interface{}
}

// VisitObjectParams params
type VisitObjectParams struct {
	Context    context.Context
	Config     *graphql.ObjectConfig
	Node       *ast.ObjectDefinition
	Extensions []*ast.ObjectDefinition
	Args       map[string]interface{}
}

// VisitFieldDefinitionParams params
type VisitFieldDefinitionParams struct {
	Context    context.Context
	Config     *graphql.Field
	Node       *ast.FieldDefinition
	Args       map[string]interface{}
	ParentName string
	ParentKind string
}

// VisitArgumentDefinitionParams params
type VisitArgumentDefinitionParams struct {
	Context context.Context
	Config  *graphql.ArgumentConfig
	Node    *ast.InputValueDefinition
	Args    map[string]interface{}
}

// VisitInterfaceParams params
type VisitInterfaceParams struct {
	Context context.Context
	Config  *graphql.InterfaceConfig
	Node    *ast.InterfaceDefinition
	Args    map[string]interface{}
}

// VisitUnionParams params
type VisitUnionParams struct {
	Context context.Context
	Config  *graphql.UnionConfig
	Node    *ast.UnionDefinition
	Args    map[string]interface{}
}

// VisitEnumParams params
type VisitEnumParams struct {
	Context context.Context
	Config  *graphql.EnumConfig
	Node    *ast.EnumDefinition
	Args    map[string]interface{}
}

// VisitEnumValueParams params
type VisitEnumValueParams struct {
	Context context.Context
	Config  *graphql.EnumValueConfig
	Node    *ast.EnumValueDefinition
	Args    map[string]interface{}
}

// VisitInputObjectParams params
type VisitInputObjectParams struct {
	Context context.Context
	Config  *graphql.InputObjectConfig
	Node    *ast.InputObjectDefinition
	Args    map[string]interface{}
}

// VisitInputFieldDefinitionParams params
type VisitInputFieldDefinitionParams struct {
	Context context.Context
	Config  *graphql.InputObjectFieldConfig
	Node    *ast.InputValueDefinition
	Args    map[string]interface{}
}

// SchemaDirectiveVisitorMap a map of schema directive visitors
type SchemaDirectiveVisitorMap map[string]*SchemaDirectiveVisitor

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

type applyDirectiveParams struct {
	config     interface{}
	directives []*ast.Directive
	node       interface{}
	extensions []*ast.ObjectDefinition
	parentName string
	parentKind string
}

// applies directives
func (c *registry) applyDirectives(p applyDirectiveParams) error {
	if c.directiveMap == nil {
		return nil
	}

	for _, def := range p.directives {
		name := def.Name.Value
		visitor, hasVisitor := c.directiveMap[name]
		if !hasVisitor {
			continue
		}

		directive, err := c.getDirective(name)
		if err != nil {
			return err
		}

		args, err := GetArgumentValues(directive.Args, def.Arguments, map[string]interface{}{})
		if err != nil {
			return err
		}

		switch p.config.(type) {
		case *graphql.SchemaConfig:
			if visitor.VisitSchema != nil {
				if err := visitor.VisitSchema(VisitSchemaParams{
					Context: c.ctx,
					Config:  p.config.(*graphql.SchemaConfig),
					Args:    args,
					Node:    p.node.(*ast.SchemaDefinition),
				}); err != nil {
					return err
				}
			}
		case *graphql.ScalarConfig:
			if visitor.VisitScalar != nil {
				if err := visitor.VisitScalar(VisitScalarParams{
					Context: c.ctx,
					Config:  p.config.(*graphql.ScalarConfig),
					Args:    args,
					Node:    p.node.(*ast.ScalarDefinition),
				}); err != nil {
					return err
				}
			}
		case *graphql.ObjectConfig:
			if visitor.VisitObject != nil {
				if err := visitor.VisitObject(VisitObjectParams{
					Context:    c.ctx,
					Config:     p.config.(*graphql.ObjectConfig),
					Args:       args,
					Node:       p.node.(*ast.ObjectDefinition),
					Extensions: p.extensions,
				}); err != nil {
					return err
				}
			}
		case *graphql.Field:
			if visitor.VisitFieldDefinition != nil {
				if err := visitor.VisitFieldDefinition(VisitFieldDefinitionParams{
					Context:    c.ctx,
					Config:     p.config.(*graphql.Field),
					Args:       args,
					Node:       p.node.(*ast.FieldDefinition),
					ParentName: p.parentName,
					ParentKind: p.parentKind,
				}); err != nil {
					return err
				}
			}
		case *graphql.ArgumentConfig:
			if visitor.VisitArgumentDefinition != nil {
				if err := visitor.VisitArgumentDefinition(VisitArgumentDefinitionParams{
					Context: c.ctx,
					Config:  p.config.(*graphql.ArgumentConfig),
					Args:    args,
					Node:    p.node.(*ast.InputValueDefinition),
				}); err != nil {
					return err
				}
			}
		case *graphql.InterfaceConfig:
			if visitor.VisitInterface != nil {
				if err := visitor.VisitInterface(VisitInterfaceParams{
					Context: c.ctx,
					Config:  p.config.(*graphql.InterfaceConfig),
					Args:    args,
					Node:    p.node.(*ast.InterfaceDefinition),
				}); err != nil {
					return err
				}
			}
		case *graphql.UnionConfig:
			if visitor.VisitUnion != nil {
				if err := visitor.VisitUnion(VisitUnionParams{
					Context: c.ctx,
					Config:  p.config.(*graphql.UnionConfig),
					Args:    args,
					Node:    p.node.(*ast.UnionDefinition),
				}); err != nil {
					return err
				}
			}
		case *graphql.EnumConfig:
			if visitor.VisitEnum != nil {
				if err := visitor.VisitEnum(VisitEnumParams{
					Context: c.ctx,
					Config:  p.config.(*graphql.EnumConfig),
					Args:    args,
					Node:    p.node.(*ast.EnumDefinition),
				}); err != nil {
					return err
				}
			}
		case *graphql.EnumValueConfig:
			if visitor.VisitEnumValue != nil {
				if err := visitor.VisitEnumValue(VisitEnumValueParams{
					Context: c.ctx,
					Config:  p.config.(*graphql.EnumValueConfig),
					Args:    args,
					Node:    p.node.(*ast.EnumValueDefinition),
				}); err != nil {
					return err
				}
			}
		case *graphql.InputObjectConfig:
			if visitor.VisitInputObject != nil {
				if err := visitor.VisitInputObject(VisitInputObjectParams{
					Context: c.ctx,
					Config:  p.config.(*graphql.InputObjectConfig),
					Args:    args,
					Node:    p.node.(*ast.InputObjectDefinition),
				}); err != nil {
					return err
				}
			}
		case *graphql.InputObjectFieldConfig:
			if visitor.VisitInputFieldDefinition != nil {
				if err := visitor.VisitInputFieldDefinition(VisitInputFieldDefinitionParams{
					Context: c.ctx,
					Config:  p.config.(*graphql.InputObjectFieldConfig),
					Args:    args,
					Node:    p.node.(*ast.InputValueDefinition),
				}); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
