package graphqlws

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
)

// ConnKey the connection key
var ConnKey interface{} = "conn"

// HandlerConfig config
type HandlerConfig struct {
	Logger       Logger
	Authenticate AuthenticateFunc
	Schema       graphql.Schema
	RootValue    map[string]interface{}
}

// NewHandler creates a new handler
func NewHandler(config HandlerConfig) http.Handler {
	var upgrader = websocket.Upgrader{
		CheckOrigin:  func(r *http.Request) bool { return true },
		Subprotocols: []string{"graphql-ws"},
	}

	mgr := &ChanMgr{
		conns: make(map[string]map[string]*ResultChan),
	}

	if config.Logger == nil {
		config.Logger = &noopLogger{}
	}

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// Establish a WebSocket connection
			var ws, err = upgrader.Upgrade(w, r, nil)

			// Bail out if the WebSocket connection could not be established
			if err != nil {
				config.Logger.Warnf("Failed to establish WebSocket connection", err)
				return
			}

			// Close the connection early if it doesn't implement the graphql-ws protocol
			if ws.Subprotocol() != "graphql-ws" {
				config.Logger.Warnf("Connection does not implement the GraphQL WS protocol")
				ws.Close()
				return
			}

			// Establish a GraphQL WebSocket connection
			NewConnection(ws, ConnectionConfig{
				Authenticate: config.Authenticate,
				Logger:       config.Logger,
				EventHandlers: ConnectionEventHandlers{
					Close: func(conn Connection) {
						config.Logger.Debugf("closing websocket: %s", conn.ID)
						mgr.DelConn(conn.ID())
					},
					StartOperation: func(
						conn Connection,
						opID string,
						data *StartMessagePayload,
					) []error {
						config.Logger.Debugf("start operations %s on connection %s", opID, conn.ID())

						ctx := context.WithValue(context.Background(), ConnKey, conn)
						resultChannel := graphql.Subscribe(graphql.Params{
							Schema:         config.Schema,
							RequestString:  data.Query,
							VariableValues: data.Variables,
							OperationName:  data.OperationName,
							Context:        ctx,
							RootObject:     config.RootValue,
						})

						mgr.Add(conn.ID(), opID, resultChannel)

						go func() {
							for {
								select {
								case <-ctx.Done():
									mgr.Del(conn.ID(), opID)
									return
								case res, more := <-resultChannel:
									if !more {
										return
									}

									errs := []error{}

									if res.HasErrors() {
										for _, err := range res.Errors {
											config.Logger.Debugf("subscription_error: %v", err)
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
						config.Logger.Debugf("stop operation %s on connection %s", opID, conn.ID())
						mgr.Del(conn.ID(), opID)
					},
				},
			})
		},
	)
}
