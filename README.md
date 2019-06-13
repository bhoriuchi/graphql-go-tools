# graphql-go-tools
Like apollo-tools for graphql-go

[![Documentation](https://godoc.org/github.com/bhoriuchi/graphql-go-tools?status.svg)](https://godoc.org/github.com/bhoriuchi/graphql-go-tools)

## Getting Started

```sh
go get github.com/bhoriuchi/graphql-go-tools
```

## Example

```go
package main

import (
  "fmt"
  "log"

  "github.com/bhoriuchi/graphql-go-tools"
  "github.com/graphql-go/graphql"
)

func main() {
  schema, err := tools.MakeExecutableSchema(tools.MakeExecutableSchemaConfig{
    TypeDefs: `
    directive @description(value: String!) on FIELD_DEFINITION

    type Foo {
      id: ID!
      name: String!
      description: String
    }
    
    type Query {
      foo(id: ID!): Foo @description(value: "bazqux")
    }`,
    Resolvers: &tools.ResolverMap{
      "Query": &tools.ObjectResolver{
        Fields: tools.FieldResolveMap{
          "foos": func(p graphql.ResolveParams) (interface{}, error) {
            // lookup data
            return foos, nil
          },
        },
      },
    },
    SchemaDirectives: &tools.SchemaDirectiveVisitorMap{
      "description": tools.SchemaDirectiveVisitor{
        VisitFieldDefinition: func(field *graphql.Field, args map[string]interface{}) {
          resolveFunc := field.Resolve
          field.Resolve = func(p graphql.ResolveParams) (interface{}, error) {
            result, err := resolveFunc(p)
            if err != nil {
              return result, err
            }
            data := result.(map[string]interface{})
            data.description = args["value"]
            return data, nil
          }
        },
      },
    },
  })

  if err != nil {
    log.Fatalf("Failed to build schema, error: %v", err)
  }

  params := graphql.Params{
    Schema: schema,
    RequestString: `
    query GetFoo {
      foo(id: "5cffbf1ccecefcfff659cea8") {
        description
      }
    }`,
  }

  r := graphql.Do(params)
  if r.HasErrors() {
		log.Fatalf("failed to execute graphql operation, errors: %+v", r.Errors)
	}
	rJSON, _ := json.Marshal(r)
	fmt.Printf("%s \n", rJSON) // {“data”:{“description”:”bazqux”}}
}

```