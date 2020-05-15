package tools

import (
	"testing"

	"github.com/graphql-go/graphql"
)

func TestConcatenateTypeDefs(t *testing.T) {
	config := ExecutableSchema{
		TypeDefs: []string{
			"type Query{}",
			`
			# a foo
			type Foo {
				name: String!
				description: String
			}
			
			extend type Query {
				foo: Foo
			}`,
			`
			interface Named {
				name: String!
			}
			
			type Bar implements Named {
				name: String!
				description: String
			}
			
			extend type Query {
				bar: Bar
			}`,
		},
	}

	schema, err := MakeExecutableSchema(config)
	if err != nil {
		t.Errorf("failed to make schema from concatenated TypeDefs: %v", err)
		return
	}

	// perform a query
	r := graphql.Do(graphql.Params{
		Schema: schema,
		RequestString: `query Query {
			foo {
				name
			}
			bar {
				name
			}
		}`,
	})

	if r.HasErrors() {
		t.Error(r.Errors)
		return
	}
}

func TestObjectIsTypeOf(t *testing.T) {
	config := ExecutableSchema{
		TypeDefs: []string{
			"type Query{}",
			`
			# a foo
			type A {
				name: String!
			}
			type B {
				description: String
			}
			union Foo = A | B
			
			extend type Query {
				foo: Foo
			}`,
		},
		Resolvers: map[string]Resolver{
			"A": &ObjectResolver{
				IsTypeOf: func(p graphql.IsTypeOfParams) bool {
					return true
				},
			},
			"B": &ObjectResolver{
				IsTypeOf: func(p graphql.IsTypeOfParams) bool {
					return false
				},
			},
		},
	}

	schema, err := MakeExecutableSchema(config)
	if err != nil {
		t.Errorf("failed to make schema from concatenated TypeDefs: %v", err)
		return
	}

	// perform a query
	r := graphql.Do(graphql.Params{
		Schema: schema,
		RequestString: `query Query {
			foo {
				...on A {
					name
				}
				...on B {
					description
				}
			}
		}`,
	})

	if r.HasErrors() {
		t.Error(r.Errors)
		return
	}
}

func TestUnionResolveType(t *testing.T) {
	config := ExecutableSchema{
		TypeDefs: []string{
			"type Query{}",
			`
			# a foo
			type A {
				name: String!
			}
			type B {
				description: String
			}
			union Foo = A | B
			
			extend type Query {
				foo: Foo
			}`,
		},
		Resolvers: map[string]Resolver{
			"Foo": &UnionResolver{
				ResolveType: func(p graphql.ResolveTypeParams) *graphql.Object {
					return p.Info.Schema.TypeMap()["A"].(*graphql.Object)
				},
			},
		},
	}

	schema, err := MakeExecutableSchema(config)
	if err != nil {
		t.Errorf("failed to make schema from concatenated TypeDefs: %v", err)
		return
	}

	// perform a query
	r := graphql.Do(graphql.Params{
		Schema: schema,
		RequestString: `query Query {
			foo {
				...on A {
					name
				}
				...on B {
					description
				}
			}
		}`,
	})

	if r.HasErrors() {
		t.Error(r.Errors)
		return
	}
}
