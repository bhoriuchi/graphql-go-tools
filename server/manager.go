package server

import (
	"sync"

	"github.com/graphql-go/graphql"
)

type ChanMgr struct {
	mx    sync.Mutex
	conns map[string]map[string]*ResultChan
}

type ResultChan struct {
	ch chan *graphql.Result
}

func (c *ChanMgr) Add(cid, oid string, ch chan *graphql.Result) {
	c.mx.Lock()
	defer c.mx.Unlock()

	conn, ok := c.conns[cid]
	if !ok {
		conn = make(map[string]*ResultChan)
		c.conns[cid] = conn
	}

	conn[oid] = &ResultChan{
		ch: ch,
	}
}

func (c *ChanMgr) DelConn(cid string) bool {
	c.mx.Lock()
	defer c.mx.Unlock()

	conn, ok := c.conns[cid]
	if !ok {
		return false
	}

	for oid := range conn {
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

	if _, ok := conn[oid]; !ok {
		return false
	}

	delete(conn, oid)

	if len(c.conns[cid]) == 0 {
		delete(c.conns, cid)
	}

	return true
}
