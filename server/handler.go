package server

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

// RequestOptions options
type RequestOptions struct {
	Query         string                 `json:"query" url:"query" schema:"query"`
	Variables     map[string]interface{} `json:"variables" url:"variables" schema:"variables"`
	OperationName string                 `json:"operationName" url:"operationName" schema:"operationName"`
}

// a workaround for getting`variables` as a JSON string
type requestOptionsCompatibility struct {
	Query         string `json:"query" url:"query" schema:"query"`
	Variables     string `json:"variables" url:"variables" schema:"variables"`
	OperationName string `json:"operationName" url:"operationName" schema:"operationName"`
}

func getFromForm(values url.Values) *RequestOptions {
	query := values.Get("query")
	if query != "" {
		// get variables map
		variables := make(map[string]interface{}, len(values))
		variablesStr := values.Get("variables")
		json.Unmarshal([]byte(variablesStr), &variables)

		return &RequestOptions{
			Query:         query,
			Variables:     variables,
			OperationName: values.Get("operationName"),
		}
	}

	return nil
}

// NewRequestOptions Parses a http.Request into GraphQL request options struct
func NewRequestOptions(r *http.Request) *RequestOptions {
	if reqOpt := getFromForm(r.URL.Query()); reqOpt != nil {
		return reqOpt
	}

	if r.Method != http.MethodPost {
		return &RequestOptions{}
	}

	if r.Body == nil {
		return &RequestOptions{}
	}

	// TODO: improve Content-Type handling
	contentTypeStr := r.Header.Get("Content-Type")
	contentTypeTokens := strings.Split(contentTypeStr, ";")
	contentType := contentTypeTokens[0]

	switch contentType {
	case ContentTypeGraphQL:
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return &RequestOptions{}
		}
		return &RequestOptions{
			Query: string(body),
		}
	case ContentTypeFormURLEncoded:
		if err := r.ParseForm(); err != nil {
			return &RequestOptions{}
		}

		if reqOpt := getFromForm(r.PostForm); reqOpt != nil {
			return reqOpt
		}

		return &RequestOptions{}

	case ContentTypeJSON:
		fallthrough
	default:
		var opts RequestOptions
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return &opts
		}
		err = json.Unmarshal(body, &opts)
		if err != nil {
			// Probably `variables` was sent as a string instead of an object.
			// So, we try to be polite and try to parse that as a JSON string
			var optsCompatible requestOptionsCompatibility
			json.Unmarshal(body, &optsCompatible)
			json.Unmarshal([]byte(optsCompatible.Variables), &opts.Variables)
		}
		return &opts
	}
}

// ContextHandler provides an entrypoint into executing graphQL queries with a
// user-provided context.
func (s *Server) ContextHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// get query
	opts := NewRequestOptions(r)

	// execute graphql query
	params := graphql.Params{
		Schema:         s.schema,
		RequestString:  opts.Query,
		VariableValues: opts.Variables,
		OperationName:  opts.OperationName,
		Context:        ctx,
	}
	if s.options.RootValueFunc != nil {
		params.RootObject = s.options.RootValueFunc(ctx, r)
	}
	result := graphql.Do(params)

	if formatErrorFunc := s.options.FormatErrorFunc; formatErrorFunc != nil && len(result.Errors) > 0 {
		formatted := make([]gqlerrors.FormattedError, len(result.Errors))
		for i, formattedError := range result.Errors {
			formatted[i] = formatErrorFunc(formattedError.OriginalError())
		}
		result.Errors = formatted
	}

	if s.options.GraphiQL != nil {
		acceptHeader := r.Header.Get("Accept")
		_, raw := r.URL.Query()["raw"]
		if !raw && !strings.Contains(acceptHeader, "application/json") && strings.Contains(acceptHeader, "text/html") {
			renderGraphiQL(s.options.GraphiQL, w, r, params)
			return
		}
	} else if s.options.Playground != nil {
		acceptHeader := r.Header.Get("Accept")
		_, raw := r.URL.Query()["raw"]
		if !raw && !strings.Contains(acceptHeader, "application/json") && strings.Contains(acceptHeader, "text/html") {
			renderPlayground(s.options.Playground, w, r)
			return
		}
	}

	// use proper JSON Header
	w.Header().Add("Content-Type", "application/json; charset=utf-8")

	var buff []byte
	if s.options.Pretty {
		w.WriteHeader(http.StatusOK)
		buff, _ = json.MarshalIndent(result, "", "\t")

		w.Write(buff)
	} else {
		w.WriteHeader(http.StatusOK)
		buff, _ = json.Marshal(result)

		w.Write(buff)
	}

	if s.options.ResultCallbackFunc != nil {
		s.options.ResultCallbackFunc(ctx, &params, result, buff)
	}
}

func (s *Server) WSHandler(rctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Establish a WebSocket connection
	var ws, err = s.upgrader.Upgrade(w, r, nil)

	// Bail out if the WebSocket connection could not be established
	if err != nil {
		s.log.Warnf("Failed to establish WebSocket connection", err)
		return
	}

	// Close the connection early if it doesn't implement the graphql-ws protocol
	if ws.Subprotocol() != "graphql-ws" {
		s.log.Warnf("Connection does not implement the GraphQL WS protocol")
		ws.Close()
		return
	}

	// Establish a GraphQL WebSocket connection
	NewConnection(ws, ConnectionConfig{
		Authenticate: s.options.WS.AuthenticateFunc,
		Logger:       s.log,
		EventHandlers: ConnectionEventHandlers{
			Close: func(conn Connection) {
				s.log.Debugf("closing websocket: %s", conn.ID)
				s.mgr.DelConn(conn.ID())
			},
			StartOperation: func(
				conn Connection,
				opID string,
				data *StartMessagePayload,
			) []error {
				s.log.Debugf("start operations %s on connection %s", opID, conn.ID())

				rootObject := map[string]interface{}{}
				if s.options.RootValueFunc != nil {
					rootObject = s.options.RootValueFunc(rctx, r)
				}
				ctx := context.WithValue(context.Background(), ConnKey, conn)
				resultChannel := graphql.Subscribe(graphql.Params{
					Schema:         s.schema,
					RequestString:  data.Query,
					VariableValues: data.Variables,
					OperationName:  data.OperationName,
					Context:        ctx,
					RootObject:     rootObject,
				})

				s.mgr.Add(conn.ID(), opID, resultChannel)

				go func() {
					for {
						select {
						case <-ctx.Done():
							s.mgr.Del(conn.ID(), opID)
							return
						case res, more := <-resultChannel:
							if !more {
								return
							}

							errs := []error{}

							if res.HasErrors() {
								for _, err := range res.Errors {
									s.log.Debugf("subscription_error: %v", err)
									errs = append(errs, err.OriginalError())
								}
							}

							conn.SendData(opID, &DataMessagePayload{
								Data:   res.Data,
								Errors: errs,
							})
						}
					}
				}()

				return nil
			},
			StopOperation: func(conn Connection, opID string) {
				s.log.Debugf("stop operation %s on connection %s", opID, conn.ID())
				s.mgr.Del(conn.ID(), opID)
			},
		},
	})
}
