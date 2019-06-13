package tools

import (
	"fmt"
	"testing"

	"github.com/graphql-go/graphql"
)

func TestConcatenateTypeDefs(t *testing.T) {
	config := ExecutableSchema{
		TypeDefs: []string{
			`
			# a foo
			type Foo {
				name: String!
				description: String
			}
			
			type Query {
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

	fmt.Println(r.Data)
}
