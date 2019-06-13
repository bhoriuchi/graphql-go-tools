package tools

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
)

// Passes build info down
type fieldInfo struct {
	kind  string
	name  string
	field string
}

// typeRegistry the registry holds all of the types
type typeRegistry struct {
	types                   map[string]graphql.Type
	directives              map[string]*graphql.Directive
	resolverMap             *ResolverMap
	directiveMap            *SchemaDirectiveVisitorMap
	rootQueryName           string
	rootMutationName        string
	definedMutationName     bool
	definedSubscriptionName bool
	rootSubscriptionName    string
	schemaDirectives        []*ast.Directive
}

// newTypeRegistry creates a new typeRegistry
func newTypeRegistry(resolvers *ResolverMap, directives *SchemaDirectiveVisitorMap) *typeRegistry {
	resolverMap := &ResolverMap{}
	directiveMap := &SchemaDirectiveVisitorMap{}
	if resolvers != nil {
		resolverMap = resolvers
	}
	if directives != nil {
		directiveMap = directives
	}

	return &typeRegistry{
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
		resolverMap:          resolverMap,
		directiveMap:         directiveMap,
		rootQueryName:        "Query",
		rootMutationName:     "Mutation",
		rootSubscriptionName: "Subscription",
		schemaDirectives:     []*ast.Directive{},
	}
}

// gets an object from the registry
func (c *typeRegistry) getObject(name string) (*graphql.Object, error) {
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
func (c *typeRegistry) typeArray() []graphql.Type {
	a := make([]graphql.Type, 0)
	for _, t := range c.types {
		a = append(a, t)
	}
	return a
}

// gets the field resolve function for a field
func (c *typeRegistry) getFieldResolveFn(info fieldInfo) graphql.FieldResolveFn {
	rmap := *c.resolverMap

	// validate that the type exists and it is the expected kind
	if cfg, ok := rmap[info.name]; ok && info.kind == cfg.getKind() {
		switch info.kind {
		case kinds.ObjectDefinition:
			if conf := cfg.getObject(); conf != nil {
				if fn, ok := conf.Fields[info.field]; ok {
					return fn
				}
			}
		case kinds.InterfaceDefinition:
			if conf := cfg.getInterface(); conf != nil {
				if fn, ok := conf.Fields[info.field]; ok {
					return fn
				}
			}
		}
	}
	return graphql.DefaultResolveFn
}

// Get gets a type from the registry
func (c *typeRegistry) getType(name string) (graphql.Type, error) {
	if val, ok := c.types[name]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("type %q not found", name)
}

// Set sets a graphql type in the registry
func (c *typeRegistry) setType(name string, graphqlType graphql.Type) {
	c.types[name] = graphqlType
}

// BuildTypesFromBody parses body and builds types
func (c *typeRegistry) buildTypesFromBody(body []byte) error {
	// parse the schema definition
	astDocument, parseErr := parser.Parse(parser.ParseParams{
		Source: &source.Source{
			Body: body,
			Name: "GraphQL",
		},
	})
	if parseErr != nil {
		return parseErr
	} else if err := c.buildTypesFromASTDocument(astDocument); err != nil {
		return err
	}
	return nil
}

// BuildTypesFromAST builds types from an ast
func (c *typeRegistry) buildTypesFromASTDocument(astValue *ast.Document, specific ...string) error {
	// allow conditional building of types so that they can be staged
	s := ""
	if len(specific) > 0 {
		s = specific[0]
	}

	for _, def := range astValue.Definitions {
		switch kind := def.GetKind(); kind {
		case kinds.SchemaDefinition:
			if s == "" || s == kind {
				schemaDef := def.(*ast.SchemaDefinition)
				c.schemaDirectives = schemaDef.Directives

				// get operations
				for _, opType := range schemaDef.OperationTypes {
					switch opType.Operation {
					case "query":
						c.rootQueryName = opType.Type.Name.Value
					case "mutation":
						c.rootMutationName = opType.Type.Name.Value
						c.definedMutationName = true
					case "subscription":
						c.rootSubscriptionName = opType.Type.Name.Value
						c.definedSubscriptionName = true
					}
				}
			}
		case kinds.ScalarDefinition:
			if s == "" || s == kind {
				if err := c.buildScalarFromAST(def.(*ast.ScalarDefinition)); err != nil {
					return err
				}
			}
		case kinds.EnumDefinition:
			if s == "" || s == kind {
				if err := c.buildEnumFromAST(def.(*ast.EnumDefinition)); err != nil {
					return err
				}
			}
		case kinds.InputObjectDefinition:
			if s == "" || s == kind {
				if err := c.buildInputObjectFromAST(def.(*ast.InputObjectDefinition)); err != nil {
					return err
				}
			}
		case kinds.InterfaceDefinition:
			if s == "" || s == kind {
				if err := c.buildInterfaceFromAST(def.(*ast.InterfaceDefinition)); err != nil {
					return err
				}
			}
		case kinds.UnionDefinition:
			if s == "" || s == kind {
				if err := c.buildUnionFromAST(def.(*ast.UnionDefinition)); err != nil {
					return err
				}
			}
		case kinds.ObjectDefinition:
			if s == "" || s == kind {
				if err := c.buildObjectFromAST(def.(*ast.ObjectDefinition)); err != nil {
					return err
				}
			}
		case kinds.DirectiveDefinition:
			if s == "" || s == kind {
				if err := c.buildDirectiveFromAST(def.(*ast.DirectiveDefinition)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// builds a scalar from ast
func (c *typeRegistry) buildScalarFromAST(definition *ast.ScalarDefinition) error {
	name := definition.Name.Value
	scalarConfig := graphql.ScalarConfig{
		Name: name,
	}
	if definition.Description != nil {
		scalarConfig.Description = definition.Description.Value
	}

	// attempt to add scalar functions
	if c.resolverMap != nil {
		rmap := *c.resolverMap
		if cfg, ok := rmap[name]; ok && cfg.getKind() == kinds.ScalarDefinition {
			if r := cfg.getScalar(); r != nil {
				scalarConfig.ParseLiteral = r.ParseLiteral
				scalarConfig.ParseValue = r.ParseValue
				scalarConfig.Serialize = r.Serialize
			}
		}
	}

	if err := c.applyDirectives(&scalarConfig, definition.Directives); err != nil {
		return err
	}

	c.types[name] = graphql.NewScalar(scalarConfig)
	return nil
}

// builds an enum from ast
func (c *typeRegistry) buildEnumFromAST(definition *ast.EnumDefinition) error {
	name := definition.Name.Value
	enumConfig := graphql.EnumConfig{
		Name:   name,
		Values: graphql.EnumValueConfigMap{},
	}
	if definition.Description != nil {
		enumConfig.Description = definition.Description.Value
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
func (c *typeRegistry) buildInterfaceFromAST(definition *ast.InterfaceDefinition) error {
	name := definition.Name.Value
	var fieldsThunk graphql.FieldsThunk
	fieldsThunk = func() graphql.Fields {
		fields := make(graphql.Fields)
		for _, f := range definition.Fields {
			info := fieldInfo{
				kind:  definition.GetKind(),
				name:  name,
				field: f.Name.Value,
			}
			field, err := c.buildFieldFromAST(f, info)
			if err != nil {
				return nil
			}
			fields[f.Name.Value] = field
		}
		return fields
	}

	ifaceConfig := graphql.InterfaceConfig{
		Name:   name,
		Fields: fieldsThunk,
	}
	if definition.Description != nil {
		ifaceConfig.Description = definition.Description.Value
	}

	// attempt to add resolveType function
	if c.resolverMap != nil {
		rmap := *c.resolverMap
		if cfg, ok := rmap[name]; ok && cfg.getKind() == kinds.InterfaceDefinition {
			if r := cfg.getInterface(); r != nil {
				ifaceConfig.ResolveType = r.ResolveType
			}
		}
	}

	if err := c.applyDirectives(&ifaceConfig, definition.Directives); err != nil {
		return err
	}

	c.types[name] = graphql.NewInterface(ifaceConfig)
	return nil
}

// builds a union from ast
func (c *typeRegistry) buildUnionFromAST(definition *ast.UnionDefinition) error {
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
func (c *typeRegistry) buildInputObjectFromAST(definition *ast.InputObjectDefinition) error {
	name := definition.Name.Value
	var fieldsThunk graphql.InputObjectConfigFieldMapThunk
	fieldsThunk = func() graphql.InputObjectConfigFieldMap {
		fields := make(graphql.InputObjectConfigFieldMap)
		for _, f := range definition.Fields {
			field, err := c.buildInputObjectFieldFromAST(f)
			if err != nil {
				return nil
			}
			fields[f.Name.Value] = field
		}
		return fields
	}

	inputConfig := graphql.InputObjectConfig{
		Name:   name,
		Fields: fieldsThunk,
	}

	if definition.Description != nil {
		inputConfig.Description = definition.Description.Value
	}

	if err := c.applyDirectives(&inputConfig, definition.Directives); err != nil {
		return err
	}

	c.types[name] = graphql.NewInputObject(inputConfig)
	return nil
}

// builds an input object field from an AST
func (c *typeRegistry) buildInputObjectFieldFromAST(definition *ast.InputValueDefinition) (*graphql.InputObjectFieldConfig, error) {
	t, err := c.buildComplexType(definition.Type)
	if err != nil {
		return nil, err
	}

	field := graphql.InputObjectFieldConfig{
		Type: t,
	}

	if definition.Description != nil {
		field.Description = definition.Description.Value
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
func (c *typeRegistry) buildObjectFromAST(definition *ast.ObjectDefinition) error {
	name := definition.Name.Value
	var fieldsThunk graphql.FieldsThunk
	fieldsThunk = func() graphql.Fields {
		fields := make(graphql.Fields)
		for _, f := range definition.Fields {
			info := fieldInfo{
				kind:  definition.GetKind(),
				name:  name,
				field: f.Name.Value,
			}
			field, err := c.buildFieldFromAST(f, info)
			if err != nil {
				return nil
			}
			fields[f.Name.Value] = field
		}
		return fields
	}

	objectConfig := graphql.ObjectConfig{
		Name:   name,
		Fields: fieldsThunk,
	}

	if definition.Description != nil {
		objectConfig.Description = definition.Description.Value
	}

	if len(definition.Interfaces) > 0 {
		var ifaceThunk graphql.InterfacesThunk
		ifaceThunk = func() []*graphql.Interface {
			ifaces := make([]*graphql.Interface, 0)
			for _, idef := range definition.Interfaces {
				iface, err := c.getType(idef.Name.Value)
				if err != nil {
					return nil
				}
				ifaces = append(ifaces, iface.(*graphql.Interface))
			}
			return ifaces
		}
		objectConfig.Interfaces = ifaceThunk
	}

	if err := c.applyDirectives(&objectConfig, definition.Directives); err != nil {
		return err
	}

	c.types[name] = graphql.NewObject(objectConfig)
	return nil
}

// Recursively builds a complex type
func (c typeRegistry) buildComplexType(astType ast.Type) (graphql.Type, error) {
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
func (c *typeRegistry) buildEnumValueFromAST(definition *ast.EnumValueDefinition, enumName string) (*graphql.EnumValueConfig, error) {
	var value interface{}
	if c.resolverMap != nil {
		rmap := *c.resolverMap
		if cfg, ok := rmap[enumName]; ok && cfg.getKind() == kinds.EnumDefinition {
			if r := cfg.getEnum(); r != nil {
				if val, ok := r.Values[definition.Name.Value]; ok {
					value = val
				}
			}
		}
	}

	if value == nil {
		value = definition.Name.Value
	}

	valueConfig := graphql.EnumValueConfig{
		Value: value,
	}
	if definition.Description != nil {
		valueConfig.Description = definition.Description.Value
	}

	if err := c.applyDirectives(&valueConfig, definition.Directives); err != nil {
		return nil, err
	}

	return &valueConfig, nil
}

// builds an arg from an ast
func (c *typeRegistry) buildArgFromAST(definition *ast.InputValueDefinition) (*graphql.ArgumentConfig, error) {
	t, err := c.buildComplexType(definition.Type)
	if err != nil {
		return nil, err
	}
	arg := graphql.ArgumentConfig{
		Type: t,
	}
	if definition.Description != nil {
		arg.Description = definition.Description.Value
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
func (c *typeRegistry) buildFieldFromAST(definition *ast.FieldDefinition, info fieldInfo) (*graphql.Field, error) {
	t, err := c.buildComplexType(definition.Type)
	if err != nil {
		return nil, err
	}

	field := graphql.Field{
		Name:    definition.Name.Value,
		Type:    t,
		Args:    graphql.FieldConfigArgument{},
		Resolve: c.getFieldResolveFn(info),
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

	if definition.Description != nil {
		field.Description = definition.Description.Value
	}

	if err := c.applyDirectives(&field, definition.Directives); err != nil {
		return nil, err
	}

	return &field, nil
}

// applies directives
func (c *typeRegistry) applyDirectives(target interface{}, directives []*ast.Directive) error {
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
