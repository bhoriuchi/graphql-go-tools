package tools

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
)

// gets the field resolve function for a field
func (c *registry) getFieldResolveFn(kind, typeName, fieldName string) graphql.FieldResolveFn {
	if r := c.getResolver(typeName); r != nil && kind == r.getKind() {
		switch kind {
		case kinds.ObjectDefinition:
			if fn, ok := r.(*ObjectResolver).Fields[fieldName]; ok {
				return fn
			}
		case kinds.InterfaceDefinition:
			if fn, ok := r.(*InterfaceResolver).Fields[fieldName]; ok {
				return fn
			}
		}
	}
	return graphql.DefaultResolveFn
}

// Recursively builds a complex type
func (c registry) buildComplexType(astType ast.Type) (graphql.Type, error) {
	switch kind := astType.GetKind(); kind {
	case kinds.List:
		t, err := c.buildComplexType(astType.(*ast.List).Type)
		if err != nil {
			return nil, err
		}
		return graphql.NewList(t), nil

	case kinds.NonNull:
		t, err := c.buildComplexType(astType.(*ast.NonNull).Type)
		if err != nil {
			return nil, err
		}
		return graphql.NewNonNull(t), nil

	case kinds.Named:
		t := astType.(*ast.Named)
		return c.getType(t.Name.Value)
	}

	return nil, fmt.Errorf("invalid kind")
}

// gets the description or defaults to an empty string
func getDescription(node ast.DescribableNode) string {
	if desc := node.GetDescription(); desc != nil {
		return desc.Value
	}
	return ""
}

// gets the default value or defaults to nil
func getDefaultValue(input *ast.InputValueDefinition) interface{} {
	if value := input.DefaultValue; value != nil {
		return value.GetValue()
	}
	return nil
}
