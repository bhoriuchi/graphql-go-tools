package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"

	"github.com/graphql-go/graphql"
)

// GraphiQLConfig a configuration for graphiql
type GraphiQLConfig struct {
	Version              string
	Endpoint             string
	SubscriptionEndpoint string
}

// NewDefaultGraphiQLConfig creates a new default config
func NewDefaultGraphiQLConfig() *GraphiQLConfig {
	return &GraphiQLConfig{
		Version:              GraphiqlVersion,
		Endpoint:             "",
		SubscriptionEndpoint: "",
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
func renderGraphiQL(config *GraphiQLConfig, w http.ResponseWriter, r *http.Request, params graphql.Params) {
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

	subscriptionPath := path.Join(path.Dir(r.URL.Path), "subscriptions")
	subscriptionEndpoint := fmt.Sprintf("ws://%v%s", r.Host, subscriptionPath)
	if config.SubscriptionEndpoint != "" {
		if _, err := url.ParseRequestURI(config.SubscriptionEndpoint); err == nil {
			subscriptionEndpoint = config.SubscriptionEndpoint
		} else {
			subscriptionEndpoint = path.Join(
				fmt.Sprintf("ws://%v", r.Host),
				config.SubscriptionEndpoint,
			)
		}
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

	return
}

// GraphiqlVersion is the current version of GraphiQL
const GraphiqlVersion = "0.13.2"

const graphiqlTemplate = `
{{ define "index" }}
<!--
 *  Copyright (c) 2019 GraphQL Contributors
 *  All rights reserved.
 *
 *  This source code is licensed under the license found in the
 *  LICENSE file in the root directory of this source tree.
-->
<!DOCTYPE html>
<html>
  <head>
    <style>
      body {
        height: 100%;
        margin: 0;
        width: 100%;
        overflow: hidden;
      }
      #graphiql {
        height: 100vh;
      }
    </style>

    <!--
      This GraphiQL example depends on Promise and fetch, which are available in
      modern browsers, but can be "polyfilled" for older browsers.
      GraphiQL itself depends on React DOM.
      If you do not want to rely on a CDN, you can host these files locally or
      include them directly in your favored resource bunder.
    -->
    <script src="//cdn.jsdelivr.net/es6-promise/4.0.5/es6-promise.auto.min.js"></script>
    <script src="//cdn.jsdelivr.net/fetch/0.9.0/fetch.min.js"></script>
    <script src="//cdn.jsdelivr.net/react/15.4.2/react.min.js"></script>
    <script src="//cdn.jsdelivr.net/react/15.4.2/react-dom.min.js"></script>
    <script src="//cdn.jsdelivr.net/npm/subscriptions-transport-ws@0.9.16/browser/client.js"></script>

    <!--
      These two files can be found in the npm module, however you may wish to
      copy them directly into your environment, or perhaps include them in your
      favored resource bundler.
     -->
    <link href="//cdn.jsdelivr.net/npm/graphiql@{{ .GraphiqlVersion }}/graphiql.css" rel="stylesheet" />
    <script src="//cdn.jsdelivr.net/npm/graphiql@{{ .GraphiqlVersion }}/graphiql.min.js"></script>

  </head>
  <body>
    <div id="graphiql">Loading...</div>
    <script>

      /**
       * This GraphiQL example illustrates how to use some of GraphiQL's props
       * in order to enable reading and updating the URL parameters, making
       * link sharing of queries a little bit easier.
       *
       * This is only one example of this kind of feature, GraphiQL exposes
       * various React params to enable interesting integrations.
       */

      // Parse the search string to get url parameters.
      var search = window.location.search;
      var parameters = {};
      search.substr(1).split('&').forEach(function (entry) {
        var eq = entry.indexOf('=');
        if (eq >= 0) {
          parameters[decodeURIComponent(entry.slice(0, eq))] =
            decodeURIComponent(entry.slice(eq + 1));
        }
      });

      // if variables was provided, try to format it.
      if (parameters.variables) {
        try {
          parameters.variables =
            JSON.stringify(JSON.parse(parameters.variables), null, 2);
        } catch (e) {
          // Do nothing, we want to display the invalid JSON as a string, rather
          // than present an error.
        }
      }

      // When the query and variables string is edited, update the URL bar so
      // that it can be easily shared
      function onEditQuery(newQuery) {
        parameters.query = newQuery;
        updateURL();
      }

      function onEditVariables(newVariables) {
        parameters.variables = newVariables;
        updateURL();
      }

      function onEditOperationName(newOperationName) {
        parameters.operationName = newOperationName;
        updateURL();
      }

      function updateURL() {
        var newSearch = '?' + Object.keys(parameters).filter(function (key) {
          return Boolean(parameters[key]);
        }).map(function (key) {
          return encodeURIComponent(key) + '=' +
            encodeURIComponent(parameters[key]);
        }).join('&');
        history.replaceState(null, null, newSearch);
      }

      // Defines a GraphQL fetcher using the fetch API. You're not required to
      // use fetch, and could instead implement graphQLFetcher however you like,
      // as long as it returns a Promise or Observable.
      function graphQLFetcher(graphQLParams) {
        return fetch({{ .Endpoint }}, {
          method: 'post',
          headers: {
            'Accept': 'application/json',
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(graphQLParams),
          credentials: 'include',
        }).then(function (response) {
          return response.text();
        }).then(function (responseBody) {
          try {
            return JSON.parse(responseBody);
          } catch (error) {
            return responseBody;
          }
        });
      }

      // Subscription support
      function hasSubscriptionOperation (graphQlParams) {
        return graphQlParams.query.match(/^subscription/mgi)
      };

      function subscriptionFetcher(subscriptionsClient, fallbackFetcher) {
        var activeSubscription = false;
        return function(graphQLParams) {
          if (subscriptionsClient && activeSubscription) {
            subscriptionsClient.unsubscribeAll();
          }
      
          if (subscriptionsClient && hasSubscriptionOperation(graphQLParams)) {
            activeSubscription = true;
      
            return subscriptionsClient.request({
              query: graphQLParams.query,
              variables: graphQLParams.variables,
            });
          } else {
            return fallbackFetcher(graphQLParams);
          }
        };
      }

      var subscriptionsClient = new window.SubscriptionsTransportWs.SubscriptionClient(
        {{ .SubscriptionEndpoint }},
        { reconnect: true }
      );
      
      fetcher = subscriptionFetcher(
        subscriptionsClient,
        graphQLFetcher
      );

      // Render <GraphiQL /> into the body.
      // See the README in the top level of this module to learn more about
      // how you can customize GraphiQL by providing different values or
      // additional child elements.
      ReactDOM.render(
        React.createElement(GraphiQL, {
          fetcher: fetcher,
          // query: parameters.query,
          // variables: parameters.variables,
          // operationName: parameters.operationName,
          query: {{ .QueryString }},
          response: {{ .ResultString }},
          variables: {{ .VariablesString }},
          operationName: {{ .OperationName }},
          onEditQuery: onEditQuery,
          onEditVariables: onEditVariables,
          onEditOperationName: onEditOperationName
        }),
        document.getElementById('graphiql')
      );
    </script>
  </body>
</html>
{{ end }}
`
