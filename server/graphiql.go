package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/graphql-go/graphql"
)

// GraphiqlVersion is the current version of GraphiQL
var GraphiqlVersion = "1.4.1"

type GraphiQLOptions struct {
	Version              string
	SSL                  bool
	Endpoint             string
	SubscriptionEndpoint string
}

func NewDefaultGraphiQLOptions() *GraphiQLOptions {
	return &GraphiQLOptions{
		Version: GraphiqlVersion,
	}
}

func NewDefaultSSLGraphiQLOption() *GraphiQLOptions {
	return &GraphiQLOptions{
		Version: GraphiqlVersion,
		SSL:     true,
	}
}

// graphiqlData is the page data structure of the rendered GraphiQL page
type graphiqlData struct {
	Endpoint             string
	SubscriptionEndpoint string
	GraphiqlVersion      string
	QueryString          string
	VariablesString      string
	OperationName        string
	ResultString         string
}

// renderGraphiQL renders the GraphiQL GUI
func renderGraphiQL(config *GraphiQLOptions, w http.ResponseWriter, r *http.Request, params graphql.Params) {
	t := template.New("GraphiQL")
	t, err := t.Parse(graphiqlTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create variables string
	vars, err := json.MarshalIndent(params.VariableValues, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	varsString := string(vars)
	if varsString == "null" {
		varsString = ""
	}

	// Create result string
	var resString string
	if params.RequestString == "" {
		resString = ""
	} else {
		result, err := json.MarshalIndent(graphql.Do(params), "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resString = string(result)
	}

	endpoint := r.URL.Path
	if config.Endpoint != "" {
		endpoint = config.Endpoint
	}

	wsScheme := "ws:"
	if config.SSL {
		wsScheme = "wss:"
	}

	subscriptionEndpoint := fmt.Sprintf("%s//%v%s", wsScheme, r.Host, r.URL.Path)
	if config.SubscriptionEndpoint != "" {
		subscriptionEndpoint = config.SubscriptionEndpoint
	}

	d := graphiqlData{
		GraphiqlVersion:      GraphiqlVersion,
		QueryString:          params.RequestString,
		ResultString:         resString,
		VariablesString:      varsString,
		OperationName:        params.OperationName,
		Endpoint:             endpoint,
		SubscriptionEndpoint: subscriptionEndpoint,
	}
	err = t.ExecuteTemplate(w, "index", d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

const graphiqlTemplate = `
{{ define "index" }}
<html>
  <head>
    <title>Simple GraphiQL Example</title>
    <link href="https://unpkg.com/graphiql/graphiql.min.css" rel="stylesheet" />
    <script
      crossorigin
      src="https://unpkg.com/react/umd/react.production.min.js"
    ></script>
    <script
      crossorigin
      src="https://unpkg.com/react-dom/umd/react-dom.production.min.js"
    ></script>
    <script
      crossorigin
      src="https://unpkg.com/graphiql/graphiql.js"
    ></script>
  
  </head>
  <body style="margin: 0;">
    <div id="graphiql" style="height: 100vh;"></div>
    <script>
      const subscriptionUrl = window.location.href.replace(/^http/, "ws")
      const fetcher = GraphiQL.createFetcher({
        url: window.location.href,
        subscriptionUrl: subscriptionUrl,
      });

      ReactDOM.render(
        React.createElement(GraphiQL, { fetcher: fetcher }),
        document.getElementById('graphiql'),
      );
    </script>
  </body>
</html>
{{end}}
`
