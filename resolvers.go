package tools

import (
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/kinds"
)

// Resolver interface to a resolver configuration
type Resolver interface {
	getKind() string
	getObject() *ObjectResolver
	getScalar() *ScalarResolver
	getInterface() *InterfaceResolver
	getUnion() *UnionResolver
	getEnum() *EnumResolver
}

// ResolverMap a map of resolver configurations
type ResolverMap map[string]Resolver

// FieldResolveMap map of field resolve functions
type FieldResolveMap map[string]graphql.FieldResolveFn

// ObjectResolver config for object resolver map
type ObjectResolver struct {
	IsTypeOf graphql.IsTypeOfFn
	Fields   FieldResolveMap
}

func (c *ObjectResolver) getKind() string {
	return kinds.ObjectDefinition
}
func (c *ObjectResolver) getObject() *ObjectResolver {
	return c
}
func (c *ObjectResolver) getScalar() *ScalarResolver {
	return nil
}
func (c *ObjectResolver) getInterface() *InterfaceResolver {
	return nil
}
func (c *ObjectResolver) getUnion() *UnionResolver {
	return nil
}
func (c *ObjectResolver) getEnum() *EnumResolver {
	return nil
}

// ScalarResolver config for a scalar resolve map
type ScalarResolver struct {
	Serialize    graphql.SerializeFn
	ParseValue   graphql.ParseValueFn
	ParseLiteral graphql.ParseLiteralFn
}

func (c *ScalarResolver) getKind() string {
	return kinds.ScalarDefinition
}
func (c *ScalarResolver) getObject() *ObjectResolver {
	return nil
}
func (c *ScalarResolver) getScalar() *ScalarResolver {
	return c
}
func (c *ScalarResolver) getInterface() *InterfaceResolver {
	return nil
}
func (c *ScalarResolver) getUnion() *UnionResolver {
	return nil
}
func (c *ScalarResolver) getEnum() *EnumResolver {
	return nil
}

// InterfaceResolver config for interface resolve
type InterfaceResolver struct {
	ResolveType graphql.ResolveTypeFn
	Fields      FieldResolveMap
}

func (c *InterfaceResolver) getKind() string {
	return kinds.InterfaceDefinition
}
func (c *InterfaceResolver) getObject() *ObjectResolver {
	return nil
}
func (c *InterfaceResolver) getScalar() *ScalarResolver {
	return nil
}
func (c *InterfaceResolver) getInterface() *InterfaceResolver {
	return c
}
func (c *InterfaceResolver) getUnion() *UnionResolver {
	return nil
}
func (c *InterfaceResolver) getEnum() *EnumResolver {
	return nil
}

// UnionResolver config for interface resolve
type UnionResolver struct {
	ResolveType graphql.ResolveTypeFn
}

func (c *UnionResolver) getKind() string {
	return kinds.UnionDefinition
}
func (c *UnionResolver) getObject() *ObjectResolver {
	return nil
}
func (c *UnionResolver) getScalar() *ScalarResolver {
	return nil
}
func (c *UnionResolver) getInterface() *InterfaceResolver {
	return nil
}
func (c *UnionResolver) getUnion() *UnionResolver {
	return c
}
func (c *UnionResolver) getEnum() *EnumResolver {
	return nil
}

// EnumResolver config for enum values
type EnumResolver struct {
	Values map[string]interface{}
}

func (c *EnumResolver) getKind() string {
	return kinds.EnumDefinition
}
func (c *EnumResolver) getObject() *ObjectResolver {
	return nil
}
func (c *EnumResolver) getScalar() *ScalarResolver {
	return nil
}
func (c *EnumResolver) getInterface() *InterfaceResolver {
	return nil
}
func (c *EnumResolver) getUnion() *UnionResolver {
	return nil
}
func (c *EnumResolver) getEnum() *EnumResolver {
	return c
}
