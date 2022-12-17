package rcon

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	defaultTimeout = 5 * time.Second
	Timeout        = errors.New("connection request timed out before a connection was available")
)

func NewConnectionPool(h string, p int, pw string) *ConnectionPool {
	return &ConnectionPool{
		host:         h,
		port:         p,
		pw:           pw,
		mu:           sync.Mutex{},
		idles:        map[string]*Connection{},
		maxOpenCount: 1,
		maxIdleCount: 1,
	}
}

type ConnectionPool struct {
	host         string
	port         int
	pw           string
	mu           sync.Mutex
	idles        map[string]*Connection
	numOpen      int
	maxOpenCount int
	maxIdleCount int
	queued       []request
}

type request struct {
	connChan chan *Connection
	errChan  chan error
}

// Return returns a previously gathered Connection from GetWithContext back to the pool for later use. The Connection
// might either be closed, put into a pool of "hot", idle connections or directly returned to a queued GetWithContext
// request.
func (p *ConnectionPool) Return(c *Connection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.maxIdleCount > len(p.idles) {
		if len(p.queued) != 0 {
			r := p.queued[0]
			p.queued = p.queued[1:]
			r.connChan <- c
		} else {
			p.idles[c.id] = c
		}
	} else {
		c.socket.Close()
		p.numOpen--
	}
}

// GetWithContext returns a connection from the pool. This method does not guarantee you to get a new, fresh Connection.
// A Connection might either be opened newly for this request, re-used from the pool of idle connections or one that was
// returned to the pool just now.
//
// If there are no idle connections and if the limit of open connections is already reached, the request to retrieve a
// Connection will be queued. The request might or might not be fulfilled later, once a Connection is returned to the
// pool.
//
// It is recommended to provide a context.Context with a deadline. The deadline will be the maximum time the caller is
// ok with waiting for a connection before a Timeout error is returned. If no deadline is provided in the context.Context,
// GetWithContext might wait indefinitely.
func (p *ConnectionPool) GetWithContext(ctx context.Context) (*Connection, error) {
	p.mu.Lock()

	if len(p.idles) > 0 {
		for _, c := range p.idles {
			delete(p.idles, c.id)
			p.mu.Unlock()
			return c, c.WithContext(ctx)
		}
	}

	if p.numOpen >= p.maxOpenCount {
		req := request{
			connChan: make(chan *Connection, 1),
			errChan:  make(chan error, 1),
		}

		p.queued = append(p.queued, req)
		p.mu.Unlock()

		timeout := defaultTimeout
		if d, o := ctx.Deadline(); o {
			timeout = time.Until(d)
		}

		select {
		case con := <-req.connChan:
			return con, con.WithContext(ctx)
		case err := <-req.errChan:
			return nil, err
		case <-time.After(timeout):
			return nil, Timeout
		}
	}

	p.numOpen++
	defer p.mu.Unlock()

	nc, err := p.new()
	if err != nil {
		p.numOpen--
		return nil, err
	}

	return nc, nc.WithContext(ctx)
}

func (p *ConnectionPool) new() (*Connection, error) {
	c, err := newSocket(p.host, p.port, p.pw)
	if err != nil {
		return nil, err
	}

	return &Connection{
		id:     fmt.Sprintf("%d", time.Now().UnixNano()),
		socket: c,
	}, nil
}
