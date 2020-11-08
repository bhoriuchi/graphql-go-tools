package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
)

// PlaygroundConfig playground configuration
type PlaygroundConfig struct {
	Endpoint             string
	SubscriptionEndpoint string
	Version              string
}

// NewDefaultPlaygroundConfig creates a new default config
func NewDefaultPlaygroundConfig() *PlaygroundConfig {
	return &PlaygroundConfig{
		Endpoint:             "",
		SubscriptionEndpoint: "",
		Version:              PlaygroundVersion,
	}
}

type playgroundData struct {
	PlaygroundVersion    string
	Endpoint             string
	SubscriptionEndpoint string
	SetTitle             bool
}

// renderPlayground renders the Playground GUI
func renderPlayground(config *PlaygroundConfig, w http.ResponseWriter, r *http.Request) {
	t := template.New("Playground")
	t, err := t.Parse(latestGraphQLPlayground)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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

	version := PlaygroundVersion
	if config.Version != "" {
		version = config.Version
	}

	d := playgroundData{
		PlaygroundVersion:    version,
		Endpoint:             endpoint,
		SubscriptionEndpoint: subscriptionEndpoint,
		SetTitle:             true,
	}
	err = t.ExecuteTemplate(w, "index", d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	return
}

// PlaygroundVersion the default version to use
var PlaygroundVersion = "1.7.20"

const graphcoolPlaygroundTemplate = `
{{ define "index" }}
<!--
The request to this GraphQL server provided the header "Accept: text/html"
and as a result has been presented Playground - an in-browser IDE for
exploring GraphQL.

If you wish to receive JSON, provide the header "Accept: application/json" or
add "&raw" to the end of the URL within a browser.
-->
<!DOCTYPE html>
<html>

<head>
  <meta charset=utf-8/>
  <meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
  <title>GraphQL Playground</title>
  <link rel="stylesheet" href="//cdn.jsdelivr.net/npm/graphql-playground-react@{{ .PlaygroundVersion }}/build/static/css/index.css" />
  <link rel="shortcut icon" href="//cdn.jsdelivr.net/npm/graphql-playground-react@{{ .PlaygroundVersion }}/build/favicon.png" />
  <script src="//cdn.jsdelivr.net/npm/graphql-playground-react@{{ .PlaygroundVersion }}/build/static/js/middleware.js"></script>
</head>

<body>
  <div id="root">
    <style>
      body {
        background-color: rgb(23, 42, 58);
        font-family: Open Sans, sans-serif;
        height: 90vh;
      }
      #root {
        height: 100%;
        width: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .loading {
        font-size: 32px;
        font-weight: 200;
        color: rgba(255, 255, 255, .6);
        margin-left: 20px;
      }
      img {
        width: 78px;
        height: 78px;
      }
      .title {
        font-weight: 400;
      }
    </style>
    <img src='//cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png' alt=''>
    <div class="loading"> Loading
      <span class="title">GraphQL Playground</span>
    </div>
  </div>
  <script>window.addEventListener('load', function (event) {
      GraphQLPlayground.init(document.getElementById('root'), {
        // options as 'endpoint' belong here
        endpoint: {{ .Endpoint }},
        subscriptionEndpoint: {{ .SubscriptionEndpoint }},
        setTitle: {{ .SetTitle }}
      })
    })</script>
</body>

</html>
{{ end }}
`

const latestGraphQLPlayground = `
{{ define "index" }}
<!DOCTYPE html>
<html>

<head>
  <meta charset=utf-8/>
  <meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
  <title>GraphQL Playground</title>
  <link rel="stylesheet" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
  <link rel="shortcut icon" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
  <script src="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>

<body>
  <div id="root">
    <style>
      body {
        background-color: rgb(23, 42, 58);
        font-family: Open Sans, sans-serif;
        height: 90vh;
      }

      #root {
        height: 100%;
        width: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
      }

      .loading {
        font-size: 32px;
        font-weight: 200;
        color: rgba(255, 255, 255, .6);
        margin-left: 20px;
      }

      img {
        width: 78px;
        height: 78px;
      }

      .title {
        font-weight: 400;
      }
    </style>
    <img src='//cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png' alt=''>
    <div class="loading"> Loading
      <span class="title">GraphQL Playground</span>
    </div>
  </div>
  <script>window.addEventListener('load', function (event) {
      GraphQLPlayground.init(document.getElementById('root'), {
        // options as 'endpoint' belong here
      })
    })</script>
</body>

</html>
{{ end }}
`
