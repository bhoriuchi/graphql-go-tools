package server

import (
	"context"
	"net/http"

	"github.com/bhoriuchi/graphql-go-tools/server/graphqlws"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
)

func (s *Server) newGraphQLWSConnection(ctx context.Context, r *http.Request, ws *websocket.Conn) {
	// Establish a GraphQL WebSocket connection
	graphqlws.NewConnection(ws, graphqlws.ConnectionConfig{
		Authenticate: s.options.WS.AuthenticateFunc,
		Logger:       s.log,
		EventHandlers: graphqlws.ConnectionEventHandlers{
			Close: func(conn graphqlws.Connection) {
				s.log.Debugf("closing websocket: %s", conn.ID())
				s.mgr.DelConn(conn.ID())
			},
			StartOperation: func(
				conn graphqlws.Connection,
				opID string,
				data *graphqlws.StartMessagePayload,
			) []error {
				s.log.Debugf("start operations %s on connection %s", opID, conn.ID())

				rootObject := map[string]interface{}{}
				if s.options.RootValueFunc != nil {
					rootObject = s.options.RootValueFunc(ctx, r)
				}
				ctx, cancelFunc := context.WithCancel(context.WithValue(context.Background(), ConnKey, conn))
				resultChannel := graphql.Subscribe(graphql.Params{
					Schema:         s.schema,
					RequestString:  data.Query,
					VariableValues: data.Variables,
					OperationName:  data.OperationName,
					Context:        ctx,
					RootObject:     rootObject,
				})

				s.mgr.Add(&ResultChan{
					ch:         resultChannel,
					cancelFunc: cancelFunc,
					ctx:        ctx,
					cid:        conn.ID(),
					oid:        opID,
				})

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

							conn.SendData(opID, &graphqlws.DataMessagePayload{
								Data:   res.Data,
								Errors: errs,
							})
						}
					}
				}()

				return nil
			},
			StopOperation: func(conn graphqlws.Connection, opID string) {
				s.log.Debugf("stop operation %s on connection %s", opID, conn.ID())
				s.mgr.Del(conn.ID(), opID)
			},
		},
	})
}
