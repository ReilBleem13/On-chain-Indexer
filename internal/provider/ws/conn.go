package ws

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/solana-foundation/solana-go/v2/rpc/ws"
)

type Conn struct {
	mu sync.Mutex

	url    string
	opts   Options
	client *ws.Client
}

type Options struct {
	Reconnect      bool
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	MaxRetries     int
	OnReconnect    func(attempt int)
}

type Option func(*Options)

func NewConn(url string, opts ...Option) *Conn {
	connOpts := Options{
		Reconnect:      true,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     20 * time.Second,
		MaxRetries:     3,
		OnReconnect: func(attempt int) {
			slog.Warn("OnReconnect", slog.Int("attempt", attempt))
		},
	}

	for _, opt := range opts {
		opt(&connOpts)
	}
	return &Conn{url: url, opts: connOpts}
}

func WithInitialBackoff(a time.Duration) func(*Options) {
	return func(o *Options) {
		o.InitialBackoff = a
	}
}

func WithMaxBackoff(a time.Duration) func(*Options) {
	return func(o *Options) {
		o.MaxBackoff = a
	}
}

func WithRetries(a int) func(*Options) {
	return func(o *Options) {
		o.MaxRetries = a
	}
}

func (c *Conn) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		return nil
	}

	client, err := ws.Connect(ctx, c.url)
	if err != nil {
		return fmt.Errorf("ws connect: %w", err)
	}

	c.client = client
	return nil
}

func (c *Conn) Client() *ws.Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.client
}

func (c *Conn) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
}
