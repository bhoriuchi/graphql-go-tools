package tools

import (
	"testing"

	"github.com/graphql-go/graphql"
)

func TestInterface(t *testing.T) {
	typeDefs := `
interface User {
	id: ID
	type: String
	name: String
}

type UserAccount implements User {
	id: ID
	type: String
	name: String
	username: String
}

type ServiceAccount implements User {
	id: ID
	type: String
	name: String
	client_id: String
}

type Query {
	users: [User]
}
`
	users := []map[string]interface{}{
		{
			"id":       "1",
			"type":     "user",
			"name":     "User1",
			"username": "user1",
		},
		{
			"id":        "1",
			"type":      "service",
			"name":      "Service1",
			"client_id": "1234567890",
		},
	}

	schema, err := MakeExecutableSchema(ExecutableSchema{
		TypeDefs: typeDefs,
		Resolvers: map[string]interface{}{
			"User": &InterfaceResolver{
				ResolveType: func(p graphql.ResolveTypeParams) *graphql.Object {
					value := p.Value.(map[string]interface{})
					typ := value["type"].(string)
					if typ == "user" {
						return p.Info.Schema.Type("UserAccount").(*graphql.Object)
					} else if typ == "service" {
						return p.Info.Schema.Type("ServiceAccount").(*graphql.Object)
					}

					return nil
				},
			},
			"Query": &ObjectResolver{
				Fields: FieldResolveMap{
					"users": &FieldResolve{
						Resolve: func(p graphql.ResolveParams) (interface{}, error) {
							return users, nil
						},
					},
				},
			},
		},
	})

	if err != nil {
		t.Errorf("failed to make schema: %v", err)
		return
	}

	r := graphql.Do(graphql.Params{
		Schema: schema,
		RequestString: `query {
			users {
				id
				type
				name
				... on UserAccount {
					username
				}
				... on ServiceAccount {
					client_id
				}
			}
		}`,
	})

	if r.HasErrors() {
		t.Error(r.Errors)
		return
	}

	// j, _ := json.MarshalIndent(r.Data, "", "  ")
	// fmt.Printf("%s\n", j)
}

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
		{
			"name": "foo",
			"meta": map[string]interface{}{
				"bar": "baz",
			},
		},
	}

	// make the schema
	_, err := MakeExecutableSchema(ExecutableSchema{
		TypeDefs: typeDefs,
		Resolvers: map[string]interface{}{
			"Query": &ObjectResolver{
				Fields: FieldResolveMap{
					"foos": &FieldResolve{
						Resolve: func(p graphql.ResolveParams) (interface{}, error) {
							return foos, nil
						},
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
		{
			"name":        "foo",
			"description": "a foo",
		},
	}

	// make the schema
	schema, err := MakeExecutableSchema(ExecutableSchema{
		TypeDefs: typeDefs,
		Resolvers: map[string]interface{}{
			"Query": &ObjectResolver{
				Fields: FieldResolveMap{
					"foos": &FieldResolve{
						Resolve: func(p graphql.ResolveParams) (interface{}, error) {
							return foos, nil
						},
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
