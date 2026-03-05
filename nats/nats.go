package nats

import (
	"fmt"

	"github.com/DC-TechHQ/tais-core/logger"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Config holds NATS connection parameters.
type Config struct {
	URL   string
	Token string // optional auth token
}

// Connect establishes a NATS connection and initialises JetStream.
// Returns the core connection and the JetStream context; both must be closed
// by the caller on shutdown.
func Connect(cfg Config, log *logger.Logger) (*nats.Conn, jetstream.JetStream, error) {
	opts := []nats.Option{
		nats.Name("tais-service"),
	}
	if cfg.Token != "" {
		opts = append(opts, nats.Token(cfg.Token))
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("nats: connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("nats: jetstream init: %w", err)
	}

	log.Info("nats: connected", "url", cfg.URL)
	return nc, js, nil
}

// Publish publishes a raw byte payload to the given subject on the core NATS
// connection (fire-and-forget).  Use JetStream Publish for durable delivery.
func Publish(nc *nats.Conn, subject string, data []byte) error {
	if err := nc.Publish(subject, data); err != nil {
		return fmt.Errorf("nats: publish to %q: %w", subject, err)
	}
	return nil
}

// Subscribe creates a plain NATS subscription on the given subject.
// Returns the subscription so the caller can Unsubscribe on shutdown.
func Subscribe(nc *nats.Conn, subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := nc.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("nats: subscribe to %q: %w", subject, err)
	}
	return sub, nil
}
