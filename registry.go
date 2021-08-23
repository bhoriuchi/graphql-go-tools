package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
)

var errUnresolvedDependencies = errors.New("unresolved dependencies")

// registry the registry holds all of the types
type registry struct {
	ctx              context.Context
	types            map[string]graphql.Type
	directives       map[string]*graphql.Directive
	schema           *graphql.Schema
	resolverMap      resolverMap
	directiveMap     SchemaDirectiveVisitorMap
	schemaDirectives []*ast.Directive
	document         *ast.Document
	extensions       []graphql.Extension
	unresolvedDefs   []ast.Node
	maxIterations    int
	iterations       int
	dependencyMap    DependencyMap
}

// newRegistry creates a new registry
func newRegistry(
	ctx context.Context,
	resolvers map[string]interface{},
	directiveMap SchemaDirectiveVisitorMap,
	extensions []graphql.Extension,
	document *ast.Document,
) (*registry, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	r := &registry{
		ctx: ctx,
		types: map[string]graphql.Type{
			"ID":       graphql.ID,
			"String":   graphql.String,
			"Int":      graphql.Int,
			"Float":    graphql.Float,
			"Boolean":  graphql.Boolean,
			"DateTime": graphql.DateTime,
		},
		directives: map[string]*graphql.Directive{
			"include":    graphql.IncludeDirective,
			"skip":       graphql.SkipDirective,
			"deprecated": graphql.DeprecatedDirective,
			"hide":       HideDirective,
		},
		resolverMap:      resolverMap{},
		directiveMap:     directiveMap,
		schemaDirectives: []*ast.Directive{},
		document:         document,
		extensions:       extensions,
		unresolvedDefs:   document.Definitions,
		iterations:       0,
		maxIterations:    len(document.Definitions),
	}

	// import each resolver to the correct location
	for name, resolver := range resolvers {
		if err := r.importResolver(name, resolver); err != nil {
			return nil, err
		}
	}

	return r, nil
}

// looks up a resolver by name or returns nil
func (c *registry) getResolver(name string) Resolver {
	if c.resolverMap != nil {
		if resolver, ok := c.resolverMap[name]; ok {
			return resolver
		}
	}
	return nil
}

// gets an object from the registry
func (c *registry) getObject(name string) (*graphql.Object, error) {
	obj, err := c.getType(name)
	if err != nil {
		return nil, err
	}
	switch o := obj.(type) {
	case *graphql.Object:
		return o, nil
	}
	return nil, nil
}

// converts the type map to an array
func (c *registry) typeArray() []graphql.Type {
	a := make([]graphql.Type, 0)
	for _, t := range c.types {
		a = append(a, t)
	}
	return a
}

// Get gets a type from the registry
func (c *registry) getType(name string) (graphql.Type, error) {
	if val, ok := c.types[name]; ok {
		return val, nil
	}

	if !c.willResolve(name) {
		return nil, fmt.Errorf("no definition found for type %q", name)
	}

	return nil, errUnresolvedDependencies
}

// Get gets a directive from the registry
func (c *registry) getDirective(name string) (*graphql.Directive, error) {
	if val, ok := c.directives[name]; ok {
		return val, nil
	}
	return nil, errUnresolvedDependencies
}

// gets the extensions for the current type
func (c *registry) getExtensions(name, kind string) []*ast.ObjectDefinition {
	extensions := []*ast.ObjectDefinition{}

	for _, def := range c.document.Definitions {
		if def.GetKind() == kinds.TypeExtensionDefinition {
			extDef := def.(*ast.TypeExtensionDefinition).Definition
			if extDef.Name.Value == name && extDef.GetKind() == kind {
				extensions = append(extensions, extDef)
			}
		}
	}

	return extensions
}

// imports a resolver from an interface
func (c *registry) importResolver(name string, resolver interface{}) error {
	switch res := resolver.(type) {
	case *graphql.Directive:
		// allow @ to be prefixed to a directive in the event there is a type with the same
		// name to allow both to be defined in the resolver map but strip it from the
		// directive before adding it to the registry
		name = strings.TrimLeft(name, "@")
		if _, ok := c.directives[name]; !ok {
			c.directives[name] = res
		}

	case *graphql.InputObject:
		if _, ok := c.types[name]; !ok {
			c.types[name] = res
		}

	case *graphql.Scalar:
		if _, ok := c.types[name]; !ok {
			c.types[name] = res
		}

	case *graphql.Enum:
		if _, ok := c.types[name]; !ok {
			c.types[name] = res
		}

	case *graphql.Object:
		if _, ok := c.types[name]; !ok {
			c.types[name] = res
		}

	case *graphql.Interface:
		if _, ok := c.types[name]; !ok {
			c.types[name] = res
		}

	case *graphql.Union:
		if _, ok := c.types[name]; !ok {
			c.types[name] = res
		}

	case *ScalarResolver:
		if _, ok := c.resolverMap[name]; !ok {
			c.resolverMap[name] = res
		}

	case *EnumResolver:
		if _, ok := c.resolverMap[name]; !ok {
			c.resolverMap[name] = res
		}

	case *ObjectResolver:
		if _, ok := c.resolverMap[name]; !ok {
			c.resolverMap[name] = res
		}

	case *InterfaceResolver:
		if _, ok := c.resolverMap[name]; !ok {
			c.resolverMap[name] = res
		}

	case *UnionResolver:
		if _, ok := c.resolverMap[name]; !ok {
			c.resolverMap[name] = res
		}
	default:
		return fmt.Errorf("invalid resolver type for %s", name)
	}

	return nil
}

func getNodeName(node ast.Node) string {
	switch node.GetKind() {
	case kinds.ObjectDefinition:
		return node.(*ast.ObjectDefinition).Name.Value
	case kinds.ScalarDefinition:
		return node.(*ast.ScalarDefinition).Name.Value
	case kinds.EnumDefinition:
		return node.(*ast.EnumDefinition).Name.Value
	case kinds.InputObjectDefinition:
		return node.(*ast.InputObjectDefinition).Name.Value
	case kinds.InterfaceDefinition:
		return node.(*ast.InterfaceDefinition).Name.Value
	case kinds.UnionDefinition:
		return node.(*ast.UnionDefinition).Name.Value
	case kinds.DirectiveDefinition:
		return node.(*ast.DirectiveDefinition).Name.Value
	}

	return ""
}

// determines if a node will resolve eventually or with a thunk
// false if there is no possibility
func (c *registry) willResolve(name string) bool {
	if _, ok := c.types[name]; ok {
		return true
	}
	for _, n := range c.unresolvedDefs {
		if getNodeName(n) == name {
			return true
		}
	}
	return false
}

// iteratively resolves dependencies until all types are resolved
func (c *registry) resolveDefinitions() error {
	unresolved := []ast.Node{}

	for len(c.unresolvedDefs) > 0 && c.iterations < c.maxIterations {
		c.iterations = c.iterations + 1

		for _, definition := range c.unresolvedDefs {
			switch nodeKind := definition.GetKind(); nodeKind {
			case kinds.DirectiveDefinition:
				if err := c.buildDirectiveFromAST(definition.(*ast.DirectiveDefinition)); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.ScalarDefinition:
				if err := c.buildScalarFromAST(definition.(*ast.ScalarDefinition)); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.EnumDefinition:
				if err := c.buildEnumFromAST(definition.(*ast.EnumDefinition)); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.InputObjectDefinition:
				if err := c.buildInputObjectFromAST(definition.(*ast.InputObjectDefinition)); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.ObjectDefinition:
				if err := c.buildObjectFromAST(definition.(*ast.ObjectDefinition)); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.InterfaceDefinition:
				if err := c.buildInterfaceFromAST(definition.(*ast.InterfaceDefinition)); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.UnionDefinition:
				if err := c.buildUnionFromAST(definition.(*ast.UnionDefinition)); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.SchemaDefinition:
				if err := c.buildSchemaFromAST(definition.(*ast.SchemaDefinition)); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			}
		}

		// check if everything has been resolved
		if len(unresolved) == 0 {
			return nil
		}

		// prepare the next loop
		c.unresolvedDefs = unresolved

		if c.iterations < c.maxIterations {
			unresolved = []ast.Node{}
		}
	}

	if len(unresolved) > 0 {
		names := []string{}
		for _, n := range unresolved {
			if name := getNodeName(n); name != "" {
				names = append(names, name)
			} else {
				names = append(names, n.GetKind())
			}
		}
		return fmt.Errorf("failed to resolve all type definitions: %v", names)
	}

	return nil
}
