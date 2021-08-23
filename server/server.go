package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/bhoriuchi/graphql-go-tools/server/graphqlws"
	"github.com/bhoriuchi/graphql-go-tools/server/logger"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

// Constants
const (
	ContentTypeJSON           = "application/json"
	ContentTypeGraphQL        = "application/graphql"
	ContentTypeFormURLEncoded = "application/x-www-form-urlencoded"
)

// ConnKey the connection key
var ConnKey interface{} = "conn"

type Server struct {
	schema   graphql.Schema
	log      logger.Logger
	options  *Options
	upgrader websocket.Upgrader
	mgr      *ChanMgr
}

func New(schema graphql.Schema, options *Options) *Server {
	if options.Logger == nil {
		options.Logger = &logger.NoopLogger{}
	}

	return &Server{
		schema:  schema,
		log:     options.Logger,
		options: options,
		upgrader: websocket.Upgrader{
			CheckOrigin:  func(r *http.Request) bool { return true },
			Subprotocols: []string{"graphql-ws"},
		},
		mgr: &ChanMgr{
			conns: make(map[string]map[string]*ResultChan),
		},
	}
}

type RootValueFunc func(ctx context.Context, r *http.Request) map[string]interface{}

type FormatErrorFunc func(err error) gqlerrors.FormattedError

type ContextFunc func(r *http.Request) context.Context

type ResultCallbackFunc func(ctx context.Context, params *graphql.Params, result *graphql.Result, responseBody []byte)

type Options struct {
	Pretty             bool
	RootValueFunc      RootValueFunc
	FormatErrorFunc    FormatErrorFunc
	ContextFunc        ContextFunc
	WSContextFunc      ContextFunc
	ResultCallbackFunc ResultCallbackFunc
	Logger             logger.Logger
	WS                 *WSOptions
	Playground         *PlaygroundOptions
	GraphiQL           *GraphiQLOptions
}

type WSOptions struct {
	AuthenticateFunc graphqlws.AuthenticateFunc
}

func IsWSUpgrade(r *http.Request) bool {
	connection := strings.ToLower(r.Header.Get("Connection"))
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))
	return connection == "upgrade" && upgrade == "websocket"
}

// ServeHTTP provides an entrypoint into executing graphQL queries.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if IsWSUpgrade(r) {
		s.log.Debugf("Upgrading connection to websocket")
		ctx := r.Context()
		if s.options.WSContextFunc != nil {
			ctx = s.options.WSContextFunc(r)
		}
		s.WSHandler(ctx, w, r)
	} else {
		ctx := r.Context()
		if s.options.ContextFunc != nil {
			ctx = s.options.ContextFunc(r)
		}
		s.ContextHandler(ctx, w, r)
	}
}
