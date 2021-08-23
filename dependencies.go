package tools

import (
	"fmt"

	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
)

type DependencyMap map[string]map[string]interface{}

func (r *registry) IdentifyDependencies() (DependencyMap, error) {
	m := DependencyMap{}

	// get list of initial types, all dependencies should be resolved
	for _, t := range r.types {
		m[t.Name()] = map[string]interface{}{}
	}

	for _, def := range r.unresolvedDefs {
		switch nodeKind := def.GetKind(); nodeKind {
		case kinds.DirectiveDefinition:
			if err := identifyDirectiveDependencies(m, def.(*ast.DirectiveDefinition)); err != nil {
				return nil, err
			}
		case kinds.ScalarDefinition:
			scalar := def.(*ast.ScalarDefinition)
			m[scalar.Name.Value] = map[string]interface{}{}
		case kinds.EnumDefinition:
			enum := def.(*ast.EnumDefinition)
			m[enum.Name.Value] = map[string]interface{}{}
		case kinds.InputObjectDefinition:
			if err := identifyInputDependencies(m, def.(*ast.InputObjectDefinition)); err != nil {
				return nil, err
			}
		case kinds.ObjectDefinition:
			if err := identifyObjectDependencies(m, def.(*ast.ObjectDefinition)); err != nil {
				return nil, err
			}
		case kinds.InterfaceDefinition:
			if err := identifyInterfaceDependencies(m, def.(*ast.InterfaceDefinition)); err != nil {
				return nil, err
			}
		case kinds.UnionDefinition:
			if err := identifyUnionDependencies(m, def.(*ast.UnionDefinition)); err != nil {
				return nil, err
			}
		case kinds.SchemaDefinition:
			identifySchemaDependencies(m, def.(*ast.SchemaDefinition))
		}
	}

	// attempt to resolve
	resolved := map[string]interface{}{}
	maxIteration := len(m) + 1
	count := 0

	for count <= maxIteration {
		count++
		if len(m) == 0 {
			break
		}

		for t, deps := range m {
			for dep := range deps {
				if _, ok := resolved[dep]; ok {
					delete(deps, dep)
				}
			}

			if len(deps) == 0 {
				resolved[t] = nil
				delete(m, t)
			}
		}
	}

	return m, nil
}

func isPrimitiveType(t string) bool {
	switch t {
	case "String", "Int", "Float", "Boolean", "ID":
		return true
	}
	return false
}

func identifyUnionDependencies(m DependencyMap, def *ast.UnionDefinition) error {
	name := def.Name.Value
	deps, ok := m[name]
	if !ok {
		deps = map[string]interface{}{}
	}

	for _, t := range def.Types {
		typeName, err := identifyRootType(t)
		if err != nil {
			return err
		}

		if !isPrimitiveType(typeName) {
			deps[typeName] = nil
		}
	}

	m[name] = deps
	return nil
}

func identifyInterfaceDependencies(m DependencyMap, def *ast.InterfaceDefinition) error {
	name := def.Name.Value
	deps, ok := m[name]
	if !ok {
		deps = map[string]interface{}{}
	}

	for _, field := range def.Fields {
		for _, arg := range field.Arguments {
			typeName, err := identifyRootType(arg.Type)
			if err != nil {
				return err
			}

			if !isPrimitiveType(typeName) {
				deps[typeName] = nil
			}
		}
		typeName, err := identifyRootType(field.Type)
		if err != nil {
			return err
		}

		if !isPrimitiveType(typeName) {
			deps[typeName] = nil
		}
	}

	m[name] = deps
	return nil
}

// schema dependencies
func identifySchemaDependencies(m DependencyMap, def *ast.SchemaDefinition) {
	deps, ok := m["schema"]
	if !ok {
		deps = map[string]interface{}{}
	}

	for _, op := range def.OperationTypes {
		switch op.Operation {
		case ast.OperationTypeQuery:
			deps[op.Type.Name.Value] = nil
		case ast.OperationTypeMutation:
			deps[op.Type.Name.Value] = nil
		case ast.OperationTypeSubscription:
			deps[op.Type.Name.Value] = nil
		}
	}

	m["schema"] = deps
}

func identifyRootType(astType ast.Type) (string, error) {
	switch kind := astType.GetKind(); kind {
	case kinds.List:
		t, err := identifyRootType(astType.(*ast.List).Type)
		if err != nil {
			return "", err
		}
		return t, nil
	case kinds.NonNull:
		t, err := identifyRootType(astType.(*ast.NonNull).Type)
		if err != nil {
			return "", err
		}
		return t, nil
	case kinds.Named:
		t := astType.(*ast.Named)
		return t.Name.Value, nil
	}

	return "", fmt.Errorf("unknown type %v", astType)
}

// directive dependencies
func identifyDirectiveDependencies(m DependencyMap, def *ast.DirectiveDefinition) error {
	name := "@" + def.Name.Value
	deps, ok := m[name]
	if !ok {
		deps = map[string]interface{}{}
	}

	for _, arg := range def.Arguments {
		typeName, err := identifyRootType(arg.Type)
		if err != nil {
			return err
		}
		if !isPrimitiveType(typeName) {
			deps[typeName] = nil
		}
	}

	m[name] = deps
	return nil
}

// gets input object depdendencies
func identifyInputDependencies(m DependencyMap, def *ast.InputObjectDefinition) error {
	name := def.Name.Value
	deps, ok := m[name]
	if !ok {
		deps = map[string]interface{}{}
	}

	for _, field := range def.Fields {
		typeName, err := identifyRootType(field.Type)
		if err != nil {
			return err
		}

		if !isPrimitiveType(typeName) {
			deps[typeName] = nil
		}
	}

	m[name] = deps
	return nil
}

// get object dependencies
func identifyObjectDependencies(m DependencyMap, def *ast.ObjectDefinition) error {
	name := def.Name.Value
	deps, ok := m[name]
	if !ok {
		deps = map[string]interface{}{}
	}

	for _, field := range def.Fields {
		for _, arg := range field.Arguments {
			typeName, err := identifyRootType(arg.Type)
			if err != nil {
				return err
			}

			if !isPrimitiveType(typeName) {
				deps[typeName] = nil
			}
		}
		typeName, err := identifyRootType(field.Type)
		if err != nil {
			return err
		}

		if !isPrimitiveType(typeName) {
			deps[typeName] = nil
		}
	}

	m[name] = deps
	return nil
}
