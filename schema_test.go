package tools

import (
	"testing"

	"github.com/graphql-go/graphql"
)

func TestMissingType(t *testing.T) {
	typeDefs := `
type Foo {
	name: String!
	meta: JSON
}

input Cyclic {
	name: String
	cyclic: Cyclic
}

type Query {
	foos: [Foo]
}`

	// create some data
	foos := []map[string]interface{}{
		map[string]interface{}{
			"name": "foo",
			"meta": map[string]interface{}{
				"bar": "baz",
			},
		},
	}

	// make the schema
	_, err := MakeExecutableSchema(ExecutableSchema{
		TypeDefs: typeDefs,
		Resolvers: ResolverMap{
			"Query": &ObjectResolver{
				Fields: FieldResolveMap{
					"foos": func(p graphql.ResolveParams) (interface{}, error) {
						return foos, nil
					},
				},
			},
		},
	})

	if err == nil {
		t.Error("expected undefined type error")
		return
	}
}

func TestMakeExecutableSchema(t *testing.T) {
	typeDefs := `
type Foo {
	name: String!
	description: String
}

type Query1 {
	foos(
		name: String
	): [Foo]
}

schema {
	query: Query1
}
`

	// create some data
	foos := []map[string]interface{}{
		map[string]interface{}{
			"name":        "foo",
			"description": "a foo",
		},
	}

	// make the schema
	schema, err := MakeExecutableSchema(ExecutableSchema{
		TypeDefs: typeDefs,
		Resolvers: ResolverMap{
			"Query": &ObjectResolver{
				Fields: FieldResolveMap{
					"foos": func(p graphql.ResolveParams) (interface{}, error) {
						return foos, nil
					},
				},
			},
		},
	})

	if err != nil {
		t.Error(err)
		return
	}

	// perform a query
	r := graphql.Do(graphql.Params{
		Schema: schema,
		RequestString: `query Query {
			foos(name:"foo") {
				name
				description
			}
		}`,
	})

	if r.HasErrors() {
		t.Error(r.Errors)
		return
	}
}

func TestMakeSchemaConfig(t *testing.T) {
	typeDefs := `
type Foo {
	name: String!
	description: String
}
`
	// make the schema
	config := ExecutableSchema{
		TypeDefs: typeDefs,
	}
	schemaConfig, err := config.MakeSchemaConfig()

	if err != nil {
		t.Error(err)
		return
	}
	objects := 0
	for _, gqlType := range schemaConfig.Types {
		_, err := gqlType.(*graphql.Object)
		if !err {
			continue
		}
		objects++
		// for k, field := range obj.Fields() {
		// 	println(k, field.Type.Name())
		// }
	}
	if objects != 1 {
		t.Error("MakeSchemaConfig does not maintain schema types")
	}
}
