package tools

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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
				return fn.Resolve
			}
		case kinds.InterfaceDefinition:
			if fn, ok := r.(*InterfaceResolver).Fields[fieldName]; ok {
				return fn.Resolve
			}
		}
	}
	return graphql.DefaultResolveFn
}

/*
func (c *registry) getFieldSubscribeFn(kind, typeName, fieldName string) graphql.FieldResolveFn {
	if r := c.getResolver(typeName); r != nil && kind == r.getKind() {
		switch kind {
		case kinds.ObjectDefinition:
			if fieldResolve, ok := r.(*ObjectResolver).Fields[fieldName]; ok {
				return fieldResolve.Subscribe
			}
		case kinds.InterfaceDefinition:
			if fieldResolve, ok := r.(*InterfaceResolver).Fields[fieldName]; ok {
				return fieldResolve.Subscribe
			}
		}
	}
	return nil
}
*/

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

// ReadSourceFiles reads all source files from a specified path
func ReadSourceFiles(p string, recursive ...bool) (string, error) {
	typeDefs := []string{}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}

	var readFunc = func(dirPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		switch ext := strings.ToLower(filepath.Ext(info.Name())); ext {
		case ".gql", ".graphql":
			data, err := ioutil.ReadFile(filepath.Join(dirPath, info.Name()))
			if err != nil {
				return err
			}
			typeDefs = append(typeDefs, string(data))
			return nil
		default:
			return nil
		}
	}

	if len(recursive) > 0 && recursive[0] {
		if err := filepath.Walk(abs, readFunc); err != nil {
			return "", err
		}
	} else {
		files, err := ioutil.ReadDir(abs)
		if err != nil {
			return "", err
		}
		for _, file := range files {
			if err := readFunc(abs, file, nil); err != nil {
				return "", err
			}
		}
	}

	result := strings.Join(typeDefs, "\n")
	return result, err
}

// UnaliasedPathArray gets the path array for a resolve function without aliases
func UnaliasedPathArray(info graphql.ResolveInfo) []interface{} {
	return unaliasedPathArray(info.Operation.GetSelectionSet(), info.Path.AsArray(), []interface{}{})
}

// gets the actual field path for a selection by removing aliases
func unaliasedPathArray(set *ast.SelectionSet, remaining []interface{}, current []interface{}) []interface{} {
	if len(remaining) == 0 {
		return current
	}

	for _, sel := range set.Selections {
		switch sel.(type) {
		case *ast.Field:
			field := sel.(*ast.Field)
			if field.Alias != nil && field.Alias.Value == remaining[0] {
				return unaliasedPathArray(sel.GetSelectionSet(), remaining[1:], append(current, field.Name.Value))
			} else if field.Name.Value == remaining[0] {
				return unaliasedPathArray(sel.GetSelectionSet(), remaining[1:], append(current, field.Name.Value))
			}
		}
	}
	return current
}

// GetPathFieldSubSelections gets the subselectiond for a path
func GetPathFieldSubSelections(info graphql.ResolveInfo, field ...string) (names []string, err error) {
	names = []string{}
	if len(info.FieldASTs) == 0 {
		return
	}

	fieldAST := info.FieldASTs[0]
	if fieldAST.GetSelectionSet() == nil {
		return
	}

	// get any sub selections
	for _, f := range field {
		for _, sel := range fieldAST.GetSelectionSet().Selections {
			switch sel.(type) {
			case *ast.InlineFragment:
				fragment := sel.(*ast.InlineFragment)
				for _, ss := range fragment.GetSelectionSet().Selections {
					switch ss.(type) {
					case *ast.Field:
						subField := ss.(*ast.Field)
						if subField.Name.Value == f {
							fieldAST = subField
							break
						}
					}
				}
			case *ast.Field:
				subField := sel.(*ast.Field)
				if subField.Name.Value == f {
					fieldAST = subField
					continue
				}
			}
		}
	}

	for _, sel := range fieldAST.GetSelectionSet().Selections {
		switch sel.(type) {
		case *ast.InlineFragment:
			fragment := sel.(*ast.InlineFragment)
			for _, ss := range fragment.GetSelectionSet().Selections {
				switch ss.(type) {
				case *ast.Field:
					field := ss.(*ast.Field)
					names = append(names, field.Name.Value)
				}
			}

		case *ast.Field:
			field := sel.(*ast.Field)
			names = append(names, field.Name.Value)
		}
	}

	return
}

// determines if a field is hidden
func isHiddenField(field *ast.FieldDefinition) bool {
	hide := false
	for _, dir := range field.Directives {
		if dir.Name.Value == directiveHide {
			return true
		}
	}

	return hide
}

const IntrospectionQuery = `query IntrospectionQuery {
  __schema {
    queryType {
      name
    }
    mutationType {
      name
    }
    subscriptionType {
      name
    }
    types {
      ...FullType
    }
    directives {
      name
      description
      locations
      args {
        ...InputValue
      }
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
    isDeprecated
    deprecationReason
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type {
    ...TypeRef
  }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
              }
            }
          }
        }
      }
    }
  }
}`
