package tools

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
)

// gets the field resolve function for a field
func (c *registry) getFieldResolveFn(kind, typeName, fieldName string) graphql.FieldResolveFn {
	if r := c.getResolver(typeName); r != nil && kind == r.GetKind() {
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

// builds a specific type from
func (c *registry) buildTypeFromDocument(document *ast.Document, buildKind string) error {
	for _, definition := range document.Definitions {
		nodeKind := definition.GetKind()

		// skip types not currently interested in
		if nodeKind != buildKind {
			continue
		}

		switch nodeKind {
		case kinds.DirectiveDefinition:
			if err := c.buildDirectiveFromAST(definition.(*ast.DirectiveDefinition)); err != nil {
				return err
			}
		case kinds.ScalarDefinition:
			if err := c.buildScalarFromAST(definition.(*ast.ScalarDefinition)); err != nil {
				return err
			}
		case kinds.EnumDefinition:
			if err := c.buildEnumFromAST(definition.(*ast.EnumDefinition)); err != nil {
				return err
			}
		case kinds.InputObjectDefinition:
			if err := c.buildInputObjectFromAST(definition.(*ast.InputObjectDefinition)); err != nil {
				return err
			}
		case kinds.ObjectDefinition:
			if err := c.buildObjectFromAST(definition.(*ast.ObjectDefinition)); err != nil {
				return err
			}
		case kinds.InterfaceDefinition:
			if err := c.buildInterfaceFromAST(definition.(*ast.InterfaceDefinition)); err != nil {
				return err
			}
		case kinds.UnionDefinition:
			if err := c.buildUnionFromAST(definition.(*ast.UnionDefinition)); err != nil {
				return err
			}
		case kinds.SchemaDefinition:
			if err := c.buildSchemaFromAST(definition.(*ast.SchemaDefinition)); err != nil {
				return err
			}
		}
	}

	return nil
}

// builds a scalar from ast
func (c *registry) buildScalarFromAST(definition *ast.ScalarDefinition) error {
	name := definition.Name.Value
	scalarConfig := graphql.ScalarConfig{
		Name:        name,
		Description: getDescription(definition),
	}

	if r := c.getResolver(name); r != nil && r.GetKind() == kinds.ScalarDefinition {
		scalarConfig.ParseLiteral = r.(*ScalarResolver).ParseLiteral
		scalarConfig.ParseValue = r.(*ScalarResolver).ParseValue
		scalarConfig.Serialize = r.(*ScalarResolver).Serialize
	}

	if err := c.applyDirectives(&scalarConfig, definition.Directives); err != nil {
		return err
	}

	c.types[name] = graphql.NewScalar(scalarConfig)
	return nil
}

// builds an enum from ast
func (c *registry) buildEnumFromAST(definition *ast.EnumDefinition) error {
	name := definition.Name.Value
	enumConfig := graphql.EnumConfig{
		Name:        name,
		Description: getDescription(definition),
		Values:      graphql.EnumValueConfigMap{},
	}

	// add values
	for _, value := range definition.Values {
		if value != nil {
			val, err := c.buildEnumValueFromAST(value, name)
			if err != nil {
				return err
			}
			enumConfig.Values[value.Name.Value] = val
		}
	}

	if err := c.applyDirectives(&enumConfig, definition.Directives); err != nil {
		return err
	}

	c.types[name] = graphql.NewEnum(enumConfig)
	return nil
}

// builds an interfacefrom ast
func (c *registry) buildInterfaceFromAST(definition *ast.InterfaceDefinition) error {
	name := definition.Name.Value
	var fieldsThunk graphql.FieldsThunk = func() graphql.Fields {
		fields := graphql.Fields{}
		for _, fieldDef := range definition.Fields {
			if field, err := c.buildFieldFromAST(fieldDef, definition.GetKind(), name); err == nil {
				fields[fieldDef.Name.Value] = field
			} else {
				return nil
			}
		}
		return fields
	}
	ifaceConfig := graphql.InterfaceConfig{
		Name:        name,
		Description: getDescription(definition),
		Fields:      fieldsThunk,
	}

	if r := c.getResolver(name); r != nil && r.GetKind() == kinds.InterfaceDefinition {
		ifaceConfig.ResolveType = r.(*InterfaceResolver).ResolveType
	}

	if err := c.applyDirectives(&ifaceConfig, definition.Directives); err != nil {
		return err
	}

	c.types[name] = graphql.NewInterface(ifaceConfig)
	return nil
}

// builds a union from ast
func (c *registry) buildUnionFromAST(definition *ast.UnionDefinition) error {
	name := definition.Name.Value
	unionConfig := graphql.UnionConfig{
		Name:  name,
		Types: []*graphql.Object{},
	}

	// add types
	for _, t := range definition.Types {
		if t != nil {
			object, err := c.getType(t.Name.Value)
			if err != nil {
				return err
			}
			if object != nil {
				switch object.(type) {
				case *graphql.Object:
					unionConfig.Types = append(unionConfig.Types, object.(*graphql.Object))
					continue
				}
			}
			return fmt.Errorf("build Union failed: no Object type %q found", t.Name.Value)
		}
	}

	if err := c.applyDirectives(&unionConfig, definition.Directives); err != nil {
		return err
	}

	c.types[name] = graphql.NewUnion(unionConfig)
	return nil
}

// builds an input from ast
func (c *registry) buildInputObjectFromAST(definition *ast.InputObjectDefinition) error {
	name := definition.Name.Value
	var fieldsThunk graphql.InputObjectConfigFieldMapThunk = func() graphql.InputObjectConfigFieldMap {
		fields := graphql.InputObjectConfigFieldMap{}
		for _, fieldDef := range definition.Fields {
			if field, err := c.buildInputObjectFieldFromAST(fieldDef); err == nil {
				fields[fieldDef.Name.Value] = field
			} else {
				return nil
			}
		}
		return fields
	}

	inputConfig := graphql.InputObjectConfig{
		Name:        name,
		Description: getDescription(definition),
		Fields:      fieldsThunk,
	}

	if err := c.applyDirectives(&inputConfig, definition.Directives); err != nil {
		return err
	}

	c.types[name] = graphql.NewInputObject(inputConfig)
	return nil
}

// builds an input object field from an AST
func (c *registry) buildInputObjectFieldFromAST(definition *ast.InputValueDefinition) (*graphql.InputObjectFieldConfig, error) {
	t, err := c.buildComplexType(definition.Type)
	if err != nil {
		return nil, err
	}

	field := graphql.InputObjectFieldConfig{
		Type:        t,
		Description: getDescription(definition),
	}

	if definition.DefaultValue != nil {
		field.DefaultValue = definition.DefaultValue.GetValue()
	}

	if err := c.applyDirectives(&field, definition.Directives); err != nil {
		return nil, err
	}

	return &field, nil
}

// builds an object from an AST
func (c *registry) buildObjectFromAST(definition *ast.ObjectDefinition) error {
	name := definition.Name.Value
	var ifacesThunk graphql.InterfacesThunk = func() []*graphql.Interface {
		ifaces := []*graphql.Interface{}
		for _, ifaceDef := range definition.Interfaces {
			if iface, err := c.getType(ifaceDef.Name.Value); err == nil {
				ifaces = append(ifaces, iface.(*graphql.Interface))
			} else {
				return nil
			}
		}
		return ifaces
	}

	var fieldsThunk graphql.FieldsThunk = func() graphql.Fields {
		fields := graphql.Fields{}
		for _, fieldDef := range definition.Fields {
			if field, err := c.buildFieldFromAST(fieldDef, definition.GetKind(), name); err == nil {
				fields[fieldDef.Name.Value] = field
			} else {
				return nil
			}
		}
		return fields
	}

	objectConfig := graphql.ObjectConfig{
		Name:        name,
		Description: getDescription(definition),
		Interfaces:  ifacesThunk,
		Fields:      fieldsThunk,
	}

	if err := c.applyDirectives(&objectConfig, definition.Directives); err != nil {
		return err
	}

	c.types[name] = graphql.NewObject(objectConfig)
	return nil
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

// builds an enum value from an ast
func (c *registry) buildEnumValueFromAST(definition *ast.EnumValueDefinition, enumName string) (*graphql.EnumValueConfig, error) {
	var value interface{}
	value = definition.Name.Value

	if r := c.getResolver(enumName); r != nil && r.GetKind() == kinds.EnumDefinition {
		if val, ok := r.(*EnumResolver).Values[definition.Name.Value]; ok {
			value = val
		}
	}

	valueConfig := graphql.EnumValueConfig{
		Value:       value,
		Description: getDescription(definition),
	}

	if err := c.applyDirectives(&valueConfig, definition.Directives); err != nil {
		return nil, err
	}

	return &valueConfig, nil
}

// builds an arg from an ast
func (c *registry) buildArgFromAST(definition *ast.InputValueDefinition) (*graphql.ArgumentConfig, error) {
	t, err := c.buildComplexType(definition.Type)
	if err != nil {
		return nil, err
	}
	arg := graphql.ArgumentConfig{
		Type:        t,
		Description: getDescription(definition),
	}

	if definition.DefaultValue != nil {
		arg.DefaultValue = definition.DefaultValue.GetValue()
	}

	if err := c.applyDirectives(&arg, definition.Directives); err != nil {
		return nil, err
	}

	return &arg, nil
}

// builds a field from an ast
func (c *registry) buildFieldFromAST(definition *ast.FieldDefinition, kind, typeName string) (*graphql.Field, error) {
	t, err := c.buildComplexType(definition.Type)
	if err != nil {
		return nil, err
	}

	field := graphql.Field{
		Name:        definition.Name.Value,
		Description: getDescription(definition),
		Type:        t,
		Args:        graphql.FieldConfigArgument{},
		Resolve:     c.getFieldResolveFn(kind, typeName, definition.Name.Value),
	}

	for _, arg := range definition.Arguments {
		if arg != nil {
			a, err := c.buildArgFromAST(arg)
			if err != nil {
				return nil, err
			}
			field.Args[arg.Name.Value] = a
		}
	}

	if err := c.applyDirectives(&field, definition.Directives); err != nil {
		return nil, err
	}

	return &field, nil
}

// gets the description or defaults to an empty string
func getDescription(node ast.DescribableNode) string {
	if desc := node.GetDescription(); desc != nil {
		return desc.Value
	}
	return ""
}
