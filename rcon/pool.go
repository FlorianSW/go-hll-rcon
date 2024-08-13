package rcon

import (
	"code.cloudfoundry.org/lager"
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

func NewConnectionPool(logger lager.Logger, h string, p int, pw string) *ConnectionPool {
	return &ConnectionPool{
		logger:       logger.Session("pool", lager.Data{"host": h, "port": p}),
		host:         h,
		port:         p,
		pw:           pw,
		mu:           sync.Mutex{},
		idles:        map[string]*Connection{},
		maxOpenCount: 10,
		maxIdleCount: 10,
	}
}

type ConnectionPool struct {
	logger       lager.Logger
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

// SetPoolSize sets the maximum number of open connections at any time. A request for a connection when the pool reached
// this size (and no idle connections are available) will be put into a queue and be served once a connection is returned
// to the pool. This queue is on a best effort basis and might fail based on the provided deadline to GetWithContext.
func (p *ConnectionPool) SetPoolSize(s int) {
	p.logger.Debug("set-pool-size", lager.Data{"old": p.maxOpenCount, "new": s})
	p.maxOpenCount = s
}

// SetMaxIdle sets the maximum number of idle connections in the pool. Idle connections are established connections to
// the server (warm) but are not ye/anymore used. Warm connections are preferably used to fulfill connection requests
// (GetWithContext) to reduce the overhead of opening and closing a connection on every request. Consider a high max
// idle connection to benefit from re-using connections as much as possible.
func (p *ConnectionPool) SetMaxIdle(mi int) {
	l := p.logger.Session("set-max-idle", lager.Data{"old": p.maxIdleCount, "new": mi})
	if mi > p.maxOpenCount {
		l.Info("exceeds-max-open")
		p.maxIdleCount = p.maxOpenCount
	} else {
		l.Debug("set")
		p.maxIdleCount = mi
	}
}

// Return returns a previously gathered Connection from GetWithContext back to the pool for later use. The Connection
// might either be closed, put into a pool of "hot", idle connections or directly returned to a queued GetWithContext
// request.
func (p *ConnectionPool) Return(c *Connection) {
	l := p.logger.Session("return", lager.Data{"id": c.id})
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.queued) != 0 {
		r := p.queued[0]
		l.Debug("re-using-for-queue")
		p.queued = p.queued[1:]
		r.connChan <- c
	} else if p.maxIdleCount > len(p.idles) {
		l.Debug("returning-idle")
		p.idles[c.id] = c
	} else {
		l.Debug("closing")
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
	deadline, ok := ctx.Deadline()
	l := p.logger.Session("get-with-context", lager.Data{"deadline": deadline, "hasDeadline": ok, "queued": len(p.queued), "open": p.numOpen, "idles": len(p.idles)})
	p.mu.Lock()

	if len(p.idles) > 0 {
		l.Debug("from-idle-pool")
		for _, c := range p.idles {
			delete(p.idles, c.id)
			p.mu.Unlock()
			return c, c.WithContext(ctx)
		}
	}

	if p.numOpen >= p.maxOpenCount {
		l.Debug("queue-request", lager.Data{"queued": len(p.queued), "open": p.numOpen})
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

	l.Debug("open-new", lager.Data{"queued": len(p.queued), "open": p.numOpen})
	p.numOpen++
	defer p.mu.Unlock()

	nc, err := p.new()
	if err != nil {
		p.numOpen--
		return nil, err
	}

	return nc, nc.WithContext(ctx)
}

func (p *ConnectionPool) WithConnection(ctx context.Context, f func(c *Connection)) error {
	c, err := p.GetWithContext(ctx)
	if err != nil {
		return err
	}
	defer p.Return(c)

	f(c)
	return nil
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

func (p *ConnectionPool) Shutdown() {
	p.mu.Lock()
	for _, c := range p.idles {
		c.socket.Close()
		p.numOpen--
	}
	p.mu.Unlock()
}
