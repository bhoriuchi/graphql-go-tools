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
			`type Bar {
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
				name: String!
				description: String
			}
			union Foo = A | B
			
			extend type Query {
				foo: Foo
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
		}`,
	})

	if r.HasErrors() {
		t.Error(r.Errors)
		return
	}
}
