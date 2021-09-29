package server

import (
	"context"
	"sync"

	"github.com/graphql-go/graphql"
)

type ChanMgr struct {
	mx    sync.Mutex
	conns map[string]map[string]*ResultChan
}

type ResultChan struct {
	ch         chan *graphql.Result
	cancelFunc context.CancelFunc
	ctx        context.Context
	cid        string
	oid        string
}

func (c *ChanMgr) Add(rc *ResultChan) { // Add(cid, oid string, ch chan *graphql.Result) {
	c.mx.Lock()
	defer c.mx.Unlock()

	conn, ok := c.conns[rc.cid]
	if !ok {
		conn = make(map[string]*ResultChan)
		c.conns[rc.cid] = conn
	}

	conn[rc.oid] = rc
}

func (c *ChanMgr) DelConn(cid string) bool {
	c.mx.Lock()
	defer c.mx.Unlock()

	conn, ok := c.conns[cid]
	if !ok {
		return false
	}

	for oid, rc := range conn {
		if rc.cancelFunc != nil {
			rc.cancelFunc()
		}
		delete(conn, oid)
	}

	delete(conn, cid)
	return true
}

func (c *ChanMgr) Del(cid, oid string) bool {
	c.mx.Lock()
	defer c.mx.Unlock()

	conn, ok := c.conns[cid]
	if !ok {
		return false
	}

	rc, ok := conn[oid]
	if !ok {
		return false
	}

	if rc.cancelFunc != nil {
		rc.cancelFunc()
	}
	delete(conn, oid)

	if len(c.conns[cid]) == 0 {
		delete(c.conns, cid)
	}

	return true
}
