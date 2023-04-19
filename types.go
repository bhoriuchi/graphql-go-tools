package tools

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
)

// builds a scalar from ast
func (c *registry) buildScalarFromAST(definition *ast.ScalarDefinition) error {
	name := definition.Name.Value
	scalarConfig := graphql.ScalarConfig{
		Name:        name,
		Description: getDescription(definition),
	}

	if r := c.getResolver(name); r != nil && r.getKind() == kinds.ScalarDefinition {
		scalarConfig.ParseLiteral = r.(*ScalarResolver).ParseLiteral
		scalarConfig.ParseValue = r.(*ScalarResolver).ParseValue
		scalarConfig.Serialize = r.(*ScalarResolver).Serialize
	}

	if err := c.applyDirectives(applyDirectiveParams{
		config:     &scalarConfig,
		directives: definition.Directives,
		node:       definition,
	}); err != nil {
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

	for _, value := range definition.Values {
		if value != nil {
			val, err := c.buildEnumValueFromAST(value, name)
			if err != nil {
				return err
			}
			enumConfig.Values[value.Name.Value] = val
		}
	}

	if err := c.applyDirectives(applyDirectiveParams{
		config:     &enumConfig,
		directives: definition.Directives,
		node:       definition,
	}); err != nil {
		return err
	}

	c.types[name] = graphql.NewEnum(enumConfig)
	return nil
}

// builds an enum value from an ast
func (c *registry) buildEnumValueFromAST(definition *ast.EnumValueDefinition, enumName string) (*graphql.EnumValueConfig, error) {
	var value interface{}
	value = definition.Name.Value

	if r := c.getResolver(enumName); r != nil && r.getKind() == kinds.EnumDefinition {
		if val, ok := r.(*EnumResolver).Values[definition.Name.Value]; ok {
			value = val
		}
	}

	valueConfig := graphql.EnumValueConfig{
		Value:       value,
		Description: getDescription(definition),
	}

	if err := c.applyDirectives(applyDirectiveParams{
		config:     &valueConfig,
		directives: definition.Directives,
		node:       definition,
	}); err != nil {
		return nil, err
	}

	return &valueConfig, nil
}

// builds an input from ast
func (c *registry) buildInputObjectFromAST(definition *ast.InputObjectDefinition) error {
	var fields interface{}
	name := definition.Name.Value
	inputConfig := graphql.InputObjectConfig{
		Name:        name,
		Description: getDescription(definition),
		Fields:      fields,
	}

	// use thunks only when allowed
	if _, ok := c.dependencyMap[name]; ok {
		var fields graphql.InputObjectConfigFieldMapThunk = func() graphql.InputObjectConfigFieldMap {
			fieldMap, err := c.buildInputObjectFieldMapFromAST(definition.Fields)
			if err != nil {
				return nil
			}
			return fieldMap
		}
		inputConfig.Fields = fields
	} else {
		fieldMap, err := c.buildInputObjectFieldMapFromAST(definition.Fields)
		if err != nil {
			return err
		}
		inputConfig.Fields = fieldMap
	}

	if err := c.applyDirectives(applyDirectiveParams{
		config:     &inputConfig,
		directives: definition.Directives,
		node:       definition,
	}); err != nil {
		return err
	}

	c.types[name] = graphql.NewInputObject(inputConfig)
	return nil
}

// builds an input object field map from ast
func (c *registry) buildInputObjectFieldMapFromAST(fields []*ast.InputValueDefinition) (graphql.InputObjectConfigFieldMap, error) {
	fieldMap := graphql.InputObjectConfigFieldMap{}
	for _, fieldDef := range fields {
		field, err := c.buildInputObjectFieldFromAST(fieldDef)
		if err != nil {
			return nil, err
		}
		fieldMap[fieldDef.Name.Value] = field
	}
	return fieldMap, nil
}

// builds an input object field from an AST
func (c *registry) buildInputObjectFieldFromAST(definition *ast.InputValueDefinition) (*graphql.InputObjectFieldConfig, error) {
	inputType, err := c.buildComplexType(definition.Type)
	if err != nil {
		return nil, err
	}

	defaultValue, err := getDefaultValue(definition)
	if err != nil {
		return nil, err
	}

	field := graphql.InputObjectFieldConfig{
		Type:         inputType,
		Description:  getDescription(definition),
		DefaultValue: defaultValue,
	}

	if err := c.applyDirectives(applyDirectiveParams{
		config:     &field,
		directives: definition.Directives,
		node:       definition,
	}); err != nil {
		return nil, err
	}

	return &field, nil
}

// builds an object from an AST
func (c *registry) buildObjectFromAST(definition *ast.ObjectDefinition) error {
	name := definition.Name.Value
	extensions := c.getExtensions(name, definition.GetKind())
	objectConfig := graphql.ObjectConfig{
		Name:        name,
		Description: getDescription(definition),
	}

	if _, ok := c.dependencyMap[name]; ok {
		// get interfaces thunk
		var ifaces graphql.InterfacesThunk = func() []*graphql.Interface {
			ifaceArr, err := c.buildInterfacesArrayFromAST(definition, extensions)
			if err != nil {
				return nil
			}
			return ifaceArr
		}

		// get fields thunk
		var fields graphql.FieldsThunk = func() graphql.Fields {
			fieldMap, err := c.buildFieldMapFromAST(definition.Fields, definition.GetKind(), name, extensions)
			if err != nil {
				return nil
			}
			return fieldMap
		}

		objectConfig.Interfaces = ifaces
		objectConfig.Fields = fields

	} else {
		// get interfaces
		ifaceArr, err := c.buildInterfacesArrayFromAST(definition, extensions)
		if err != nil {
			return err
		}

		// get fields
		fieldMap, err := c.buildFieldMapFromAST(definition.Fields, definition.GetKind(), name, extensions)
		if err != nil {
			return err
		}

		objectConfig.Interfaces = ifaceArr
		objectConfig.Fields = fieldMap
	}

	// set IsTypeOf from resolvers
	if r := c.getResolver(name); r != nil {
		if resolver, ok := r.(*ObjectResolver); ok {
			objectConfig.IsTypeOf = resolver.IsTypeOf
		}
	}

	// update description from extensions if none
	for _, extDef := range extensions {
		if objectConfig.Description != "" {
			break
		}
		objectConfig.Description = getDescription(extDef)
	}

	// create a combined directives array
	directiveDefs := append([]*ast.Directive{}, definition.Directives...)
	for _, extDef := range extensions {
		directiveDefs = append(directiveDefs, extDef.Directives...)
	}

	if err := c.applyDirectives(applyDirectiveParams{
		config:     &objectConfig,
		directives: directiveDefs,
		extensions: extensions,
		node:       definition,
	}); err != nil {
		return err
	}

	c.types[name] = graphql.NewObject(objectConfig)
	return nil
}

func (c *registry) buildInterfacesArrayFromAST(definition *ast.ObjectDefinition, extensions []*ast.ObjectDefinition) ([]*graphql.Interface, error) {
	imap := map[string]bool{}
	ifaces := []*graphql.Interface{}

	// build list of interfaces and append extensions
	ifaceDefs := append([]*ast.Named{}, definition.Interfaces...)
	for _, extDef := range extensions {
		ifaceDefs = append(ifaceDefs, extDef.Interfaces...)
	}

	// add defined interfaces
	for _, ifaceDef := range ifaceDefs {
		if _, ok := imap[ifaceDef.Name.Value]; !ok {
			iface, err := c.getType(ifaceDef.Name.Value)
			if err != nil {
				return nil, err
			}
			ifaces = append(ifaces, iface.(*graphql.Interface))
			imap[ifaceDef.Name.Value] = true
		}
	}

	return ifaces, nil
}

func (c *registry) buildFieldMapFromAST(fields []*ast.FieldDefinition, kind, typeName string, extensions []*ast.ObjectDefinition) (graphql.Fields, error) {
	fieldMap := graphql.Fields{}

	// build list of fields and append extensions
	fieldDefs := append([]*ast.FieldDefinition{}, fields...)
	for _, extDef := range extensions {
		fieldDefs = append(fieldDefs, extDef.Fields...)
	}

	// add defined fields
	for _, fieldDef := range fieldDefs {
		if _, ok := fieldMap[fieldDef.Name.Value]; !ok {
			if field, err := c.buildFieldFromAST(fieldDef, kind, typeName); err == nil {
				if !isHiddenField(fieldDef) {
					fieldMap[fieldDef.Name.Value] = field
				}
			} else {
				return nil, err
			}
		}
	}

	return fieldMap, nil
}

// builds an interfacefrom ast
func (c *registry) buildInterfaceFromAST(definition *ast.InterfaceDefinition) error {
	extensions := []*ast.ObjectDefinition{}
	name := definition.Name.Value
	ifaceConfig := graphql.InterfaceConfig{
		Name:        name,
		Description: getDescription(definition),
	}

	if _, ok := c.dependencyMap[name]; ok {
		var fields graphql.FieldsThunk = func() graphql.Fields {
			fieldMap, err := c.buildFieldMapFromAST(definition.Fields, definition.GetKind(), name, extensions)
			if err != nil {
				return nil
			}
			return fieldMap
		}
		ifaceConfig.Fields = fields
	} else {
		fieldMap, err := c.buildFieldMapFromAST(definition.Fields, definition.GetKind(), name, extensions)
		if err != nil {
			return err
		}
		ifaceConfig.Fields = fieldMap
	}

	if r := c.getResolver(name); r != nil && r.getKind() == kinds.InterfaceDefinition {
		ifaceConfig.ResolveType = r.(*InterfaceResolver).ResolveType
	}

	if err := c.applyDirectives(applyDirectiveParams{
		config:     &ifaceConfig,
		directives: definition.Directives,
		node:       definition,
	}); err != nil {
		return err
	}

	c.types[name] = graphql.NewInterface(ifaceConfig)
	return nil
}

// builds an arg from an ast
func (c *registry) buildArgFromAST(definition *ast.InputValueDefinition) (*graphql.ArgumentConfig, error) {
	inputType, err := c.buildComplexType(definition.Type)
	if err != nil {
		return nil, err
	}

	defaultValue, err := getDefaultValue(definition)
	if err != nil {
		return nil, err
	}

	arg := graphql.ArgumentConfig{
		Type:         inputType,
		Description:  getDescription(definition),
		DefaultValue: defaultValue,
	}

	if err := c.applyDirectives(applyDirectiveParams{
		config:     &arg,
		directives: definition.Directives,
		node:       definition,
	}); err != nil {
		return nil, err
	}

	return &arg, nil
}

// builds a field from an ast
func (c *registry) buildFieldFromAST(definition *ast.FieldDefinition, kind, typeName string) (*graphql.Field, error) {
	fieldType, err := c.buildComplexType(definition.Type)
	if err != nil {
		return nil, err
	}

	field := graphql.Field{
		Name:        definition.Name.Value,
		Description: getDescription(definition),
		Type:        fieldType,
		Args:        graphql.FieldConfigArgument{},
		Resolve:     c.getFieldResolveFn(kind, typeName, definition.Name.Value),
		Subscribe:   c.getFieldSubscribeFn(kind, typeName, definition.Name.Value),
	}

	for _, arg := range definition.Arguments {
		if arg != nil {
			argValue, err := c.buildArgFromAST(arg)
			if err != nil {
				return nil, err
			}
			field.Args[arg.Name.Value] = argValue
		}
	}

	if err := c.applyDirectives(applyDirectiveParams{
		config:     &field,
		directives: definition.Directives,
		node:       definition,
		parentName: typeName,
		parentKind: kind,
	}); err != nil {
		return nil, err
	}

	return &field, nil
}

// builds a union from ast
func (c *registry) buildUnionFromAST(definition *ast.UnionDefinition) error {
	name := definition.Name.Value
	types := []*graphql.Object{}

	// add types
	for _, unionType := range definition.Types {
		object, err := c.getType(unionType.Name.Value)
		if err != nil {
			return err
		}

		if o, ok := object.(*graphql.Object); ok {
			types = append(types, o)
			continue
		}

		return fmt.Errorf("build Union failed: no Object type %q found", unionType.Name.Value)
	}

	unionConfig := &graphql.UnionConfig{
		Name:        name,
		Types:       types,
		Description: getDescription(definition),
	}

	// set ResolveType from resolvers
	if r := c.getResolver(name); r != nil {
		if resolver, ok := r.(*UnionResolver); ok {
			unionConfig.ResolveType = resolver.ResolveType
		}
	}

	if err := c.applyDirectives(applyDirectiveParams{
		config:     &unionConfig,
		directives: definition.Directives,
		node:       definition,
	}); err != nil {
		return err
	}

	c.types[name] = graphql.NewUnion(*unionConfig)
	return nil
}
