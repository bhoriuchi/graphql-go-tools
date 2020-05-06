package tools

import (
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
)

var errUnresolvedDependencies = errors.New("unresolved dependencies")

// registry the registry holds all of the types
type registry struct {
	types            map[string]graphql.Type
	directives       map[string]*graphql.Directive
	schema           *graphql.Schema
	schemaConfig     *graphql.SchemaConfig
	resolverMap      ResolverMap
	directiveMap     SchemaDirectiveVisitorMap
	schemaDirectives []*ast.Directive
	document         *ast.Document
	extensions       []graphql.Extension
	unresolvedDefs   []ast.Node
	maxIterations    int
	iterations       int
}

// newRegistry creates a new registry
func newRegistry(
	resolvers map[string]Resolver,
	directiveMap SchemaDirectiveVisitorMap,
	extensions []graphql.Extension,
	document *ast.Document,
) *registry {
	r := &registry{
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
		},
		resolverMap:      ResolverMap{},
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
		r.importResolver(name, resolver)
	}

	return r
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
	switch obj.(type) {
	case *graphql.Object:
		return obj.(*graphql.Object), nil
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

// Set sets a graphql type in the registry
func (c *registry) setType(name string, graphqlType graphql.Type) {
	c.types[name] = graphqlType
}

// Get gets a directive from the registry
func (c *registry) getDirective(name string) (*graphql.Directive, error) {
	if val, ok := c.directives[name]; ok {
		return val, nil
	}
	return nil, errUnresolvedDependencies
}

// Set sets a graphql directive in the registry
func (c *registry) setDirective(name string, graphqlDirective *graphql.Directive) {
	c.directives[name] = graphqlDirective
}

// gets the extensions for the current type
func (c *registry) getExtensions(name, kind string) []interface{} {
	extensions := []interface{}{}

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
func (c *registry) importResolver(name string, resolver interface{}) {
	switch resolver.(type) {
	case *graphql.Directive:
		if _, ok := c.directives[name]; !ok {
			c.directives[name] = resolver.(*graphql.Directive)
		}

	case *graphql.InputObject:
		if _, ok := c.types[name]; !ok {
			c.types[name] = resolver.(*graphql.InputObject)
		}

	case *graphql.Scalar:
		if _, ok := c.types[name]; !ok {
			c.types[name] = resolver.(*graphql.Scalar)
		}

	case *graphql.Enum:
		if _, ok := c.types[name]; !ok {
			c.types[name] = resolver.(*graphql.Enum)
		}

	case *graphql.Object:
		if _, ok := c.types[name]; !ok {
			c.types[name] = resolver.(*graphql.Object)
		}

	case *graphql.Interface:
		if _, ok := c.types[name]; !ok {
			c.types[name] = resolver.(*graphql.Interface)
		}

	case *graphql.Union:
		if _, ok := c.types[name]; !ok {
			c.types[name] = resolver.(*graphql.Union)
		}

	case *ScalarResolver:
		if _, ok := c.resolverMap[name]; !ok {
			c.resolverMap[name] = resolver.(*ScalarResolver)
		}

	case *EnumResolver:
		if _, ok := c.resolverMap[name]; !ok {
			c.resolverMap[name] = resolver.(*EnumResolver)
		}

	case *ObjectResolver:
		if _, ok := c.resolverMap[name]; !ok {
			c.resolverMap[name] = resolver.(*ObjectResolver)
		}

	case *InterfaceResolver:
		if _, ok := c.resolverMap[name]; !ok {
			c.resolverMap[name] = resolver.(*InterfaceResolver)
		}

	case *UnionResolver:
		if _, ok := c.resolverMap[name]; !ok {
			c.resolverMap[name] = resolver.(*UnionResolver)
		}
	}
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
		allowThunks := c.iterations == c.maxIterations

		for _, definition := range c.unresolvedDefs {
			switch nodeKind := definition.GetKind(); nodeKind {
			case kinds.DirectiveDefinition:
				if err := c.buildDirectiveFromAST(definition.(*ast.DirectiveDefinition), allowThunks); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.ScalarDefinition:
				if err := c.buildScalarFromAST(definition.(*ast.ScalarDefinition), allowThunks); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.EnumDefinition:
				if err := c.buildEnumFromAST(definition.(*ast.EnumDefinition), allowThunks); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.InputObjectDefinition:
				if err := c.buildInputObjectFromAST(definition.(*ast.InputObjectDefinition), allowThunks); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.ObjectDefinition:
				if err := c.buildObjectFromAST(definition.(*ast.ObjectDefinition), allowThunks); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.InterfaceDefinition:
				if err := c.buildInterfaceFromAST(definition.(*ast.InterfaceDefinition), allowThunks); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.UnionDefinition:
				if err := c.buildUnionFromAST(definition.(*ast.UnionDefinition), allowThunks); err != nil {
					if err == errUnresolvedDependencies {
						unresolved = append(unresolved, definition)
					} else {
						return err
					}
				}
			case kinds.SchemaDefinition:
				if err := c.buildSchemaFromAST(definition.(*ast.SchemaDefinition), allowThunks); err != nil {
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
