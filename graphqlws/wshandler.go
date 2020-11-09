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
var ConnKey interface{} = "graphqlWSConn"

// stores information about the channel and context of the subscription
type operationStore struct {
	cancelFunc context.CancelFunc
	resultChan chan *graphql.Result
}

// cancels the operation context and closes the result channel
func (c *operationStore) cancel() {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	if _, more := <-c.resultChan; more {
		close(c.resultChan)
	}
}

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

	var connections = make(map[string]map[string]*operationStore)

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

					// Close cancels any operations and closes the connection
					Close: func(conn Connection) {
						logger.WithFields(logrus.Fields{
							"conn": conn.ID(),
						}).Debug("Closing connection")

						// cancel any operations and clean up the connection
						for opID := range connections[conn.ID()] {
							store, ok := connections[conn.ID()][opID]
							if ok && store != nil {
								store.cancel()
							}
							delete(connections[conn.ID()], opID)
						}

						delete(connections, conn.ID())
					},

					// StartOperation performs the subsctition request
					StartOperation: func(
						conn Connection,
						opID string,
						data *StartMessagePayload,
					) []error {
						logger.WithFields(logrus.Fields{
							"conn": conn.ID(),
							"op":   opID,
						}).Debug("Start operation")

						// create a new cancellable context and store the connection there
						// the connection can be used to access the current authentication key
						ctx, cancelFunc := context.WithCancel(
							context.WithValue(context.Background(), ConnKey, conn),
						)

						// perform subscribe operation
						resultChan := graphql.Subscribe(graphql.Params{
							Schema:         config.Schema,
							RequestString:  data.Query,
							VariableValues: data.Variables,
							OperationName:  data.OperationName,
							Context:        ctx,
							RootObject:     config.RootValue,
						})

						// create a new operation store with the cancelFunc and result channel
						connections[conn.ID()][opID] = &operationStore{
							cancelFunc: cancelFunc,
							resultChan: resultChan,
						}

						// listen for messages
						go func() {
							for {
								select {
								case <-ctx.Done():
									return
								case res, more := <-resultChan:
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

					// Stops and cleans up the subscription
					StopOperation: func(conn Connection, opID string) {
						logger.WithFields(logrus.Fields{
							"conn": conn.ID(),
							"op":   opID,
						}).Debug("Stop operation")

						// get the store, and if found attempt to cancel the operation and close the channel
						store, ok := connections[conn.ID()][opID]
						if ok && store != nil {
							store.cancel()
						}

						// clean up the operation from the connections
						delete(connections[conn.ID()], opID)
					},
				},
			})

			// create a new operation hash for the connection
			connections[conn.ID()] = map[string]*operationStore{}
		},
	)
}
