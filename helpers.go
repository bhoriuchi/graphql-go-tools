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

// Recursively builds a complex type
func (c *registry) buildComplexType(astType ast.Type) (graphql.Type, error) {
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

func parseDefaultValue(inputType ast.Type, value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch t := inputType.(type) {
	// non-null call parse on type
	case *ast.NonNull:
		return parseDefaultValue(t.Type, value)

	// list parse each item in the list
	case *ast.List:
		switch a := value.(type) {
		case []ast.Value:
			arr := []interface{}{}
			for _, v := range a {
				val, err := parseDefaultValue(t.Type, v.GetValue())
				if err != nil {
					return nil, err
				}

				arr = append(arr, val)
			}
			return arr, nil
		}

	// parse the specific type
	case *ast.Named:
		switch t.Name.Value {
		case "Int":
			value = graphql.Int.ParseValue(value)
		case "Float":
			value = graphql.Float.ParseValue(value)
		case "Boolean":
			value = graphql.Boolean.ParseValue(value)
		case "ID":
			value = graphql.ID.ParseValue(value)
		case "String":
			value = graphql.String.ParseValue(value)
		}
	}

	return value, nil
}

// gets the default value or defaults to nil
func getDefaultValue(input *ast.InputValueDefinition) (interface{}, error) {
	if input.DefaultValue == nil {
		return nil, nil
	}

	defaultValue, err := parseDefaultValue(input.Type, input.DefaultValue.GetValue())
	if err != nil {
		return nil, err
	}

	return defaultValue, err
}

// ReadSourceFiles reads all source files from a specified path
func ReadSourceFiles(p string, recursive ...bool) (string, error) {
	typeDefs := []string{}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}

	var readFunc = func(p string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		switch ext := strings.ToLower(filepath.Ext(info.Name())); ext {
		case ".gql", ".graphql":
			data, err := ioutil.ReadFile(p)
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
		switch field := sel.(type) {
		case *ast.Field:
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
			switch fragment := sel.(type) {
			case *ast.InlineFragment:
				for _, ss := range fragment.GetSelectionSet().Selections {
					switch subField := ss.(type) {
					case *ast.Field:
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
		switch fragment := sel.(type) {
		case *ast.InlineFragment:
			for _, ss := range fragment.GetSelectionSet().Selections {
				switch field := ss.(type) {
				case *ast.Field:
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

// Merges object definitions
func MergeExtensions(obj *ast.ObjectDefinition, extensions ...*ast.ObjectDefinition) *ast.ObjectDefinition {
	merged := &ast.ObjectDefinition{
		Kind:        obj.Kind,
		Loc:         obj.Loc,
		Name:        obj.Name,
		Description: obj.Description,
		Interfaces:  append([]*ast.Named{}, obj.Interfaces...),
		Directives:  append([]*ast.Directive{}, obj.Directives...),
		Fields:      append([]*ast.FieldDefinition{}, obj.Fields...),
	}

	for _, ext := range extensions {
		merged.Interfaces = append(merged.Interfaces, ext.Interfaces...)
		merged.Directives = append(merged.Directives, ext.Directives...)
		merged.Fields = append(merged.Fields, ext.Fields...)
	}

	return merged
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
