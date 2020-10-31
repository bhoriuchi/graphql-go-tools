package graphqlws

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

// ConnKey the connection key
var ConnKey interface{} = "conn"

// SetLogger sets the logger
func SetLogger(externalLogger *logrus.Logger) {
	logger = externalLogger
}

// HandlerConfig config
type HandlerConfig struct {
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

	var connections = make(map[string]map[string]chan *graphql.Result)

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// Establish a WebSocket connection
			var ws, err = upgrader.Upgrade(w, r, nil)

			// Bail out if the WebSocket connection could not be established
			if err != nil {
				logger.Warn("Failed to establish WebSocket connection", err)
				return
			}

			// Close the connection early if it doesn't implement the graphql-ws protocol
			if ws.Subprotocol() != "graphql-ws" {
				logger.Warn("Connection does not implement the GraphQL WS protocol")
				ws.Close()
				return
			}

			// Establish a GraphQL WebSocket connection
			conn := NewConnection(ws, ConnectionConfig{
				Authenticate: config.Authenticate,
				EventHandlers: ConnectionEventHandlers{
					Close: func(conn Connection) {
						logger.WithFields(logrus.Fields{
							"conn": conn.ID(),
						}).Debug("Closing connection")

						for opID := range connections[conn.ID()] {
							iterator, ok := connections[conn.ID()][opID]
							if ok && iterator != nil {
								close(connections[conn.ID()][opID])
							}
							delete(connections[conn.ID()], opID)
						}

						delete(connections, conn.ID())
					},
					StartOperation: func(
						conn Connection,
						opID string,
						data *StartMessagePayload,
					) []error {
						logger.WithFields(logrus.Fields{
							"conn": conn.ID(),
							"op":   opID,
						}).Debug("Start operation")

						ctx := context.WithValue(context.Background(), ConnKey, conn)
						resultChannel := graphql.Subscribe(graphql.Params{
							Schema:         config.Schema,
							RequestString:  data.Query,
							VariableValues: data.Variables,
							OperationName:  data.OperationName,
							Context:        ctx,
							RootObject:     config.RootValue,
						})

						connections[conn.ID()][opID] = resultChannel

						go func() {
							for {
								select {
								case <-ctx.Done():
									return
								case res, more := <-resultChannel:
									if !more {
										return
									}

									errs := []error{}

									if res.HasErrors() {
										for _, err := range res.Errors {
											logger.Debugf("SubscriptionError: %v", err)
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
						logger.WithFields(logrus.Fields{
							"conn": conn.ID(),
							"op":   opID,
						}).Debug("Stop operation")

						iterator, ok := connections[conn.ID()][opID]
						if ok && iterator != nil {
							close(connections[conn.ID()][opID])
						}
						delete(connections[conn.ID()], opID)
					},
				},
			})

			connections[conn.ID()] = map[string]chan *graphql.Result{}
		},
	)
}
