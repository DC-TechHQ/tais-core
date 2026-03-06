package nats

import (
	"encoding/json"
	"fmt"

	"github.com/DC-TechHQ/tais-core/logger"
	"github.com/nats-io/nats.go"
)

// Config holds NATS connection parameters.
type Config struct {
	// URL is the NATS server address, e.g. "nats://nats:4222".
	URL string
}

// Connect establishes a NATS connection with automatic reconnect, then
// initialises a JetStream context. Returns error if the initial connection fails.
//
// Graceful shutdown — always drain before closing:
//
//	defer nc.Drain()
func Connect(cfg Config, log *logger.Logger) (*nats.Conn, nats.JetStreamContext, error) {
	opts := []nats.Option{
		nats.MaxReconnects(-1), // reconnect indefinitely
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Warn("nats: reconnected", "url", nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				log.Error("nats: disconnected", "error", err)
			}
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			log.Warn("nats: connection closed")
		}),
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("nats: connect: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("nats: jetstream init: %w", err)
	}

	log.Info("nats: connected", "url", cfg.URL)
	return nc, js, nil
}

// Publish marshals payload to JSON and publishes it to the given NATS subject
// via JetStream. Use JetStream publish for durable, acknowledged delivery.
func Publish(js nats.JetStreamContext, subject string, payload any, log *logger.Logger) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("nats: marshal payload for %q: %w", subject, err)
	}

	if _, err := js.Publish(subject, data); err != nil {
		log.Error("nats: publish failed", "subject", subject, "error", err)
		return fmt.Errorf("nats: publish to %q: %w", subject, err)
	}
	return nil
}

// Subscribe creates a durable push consumer on the given subject.
// The consumer name ensures the broker tracks delivery state across service
// restarts and reconnects — messages are never lost.
//
// The handler is wrapped with panic recovery: a panicking handler NAKs the
// message (so it gets redelivered) instead of crashing the service.
//
// Call nc.Drain() on shutdown to flush in-flight messages before exit.
func Subscribe(
	js nats.JetStreamContext,
	subject, consumer string,
	handler func(*nats.Msg),
	log *logger.Logger,
) {
	_, err := js.Subscribe(subject, func(msg *nats.Msg) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("nats: handler panic",
					"subject", subject,
					"consumer", consumer,
					"panic", fmt.Sprintf("%v", r),
				)
				_ = msg.Nak()
			}
		}()
		handler(msg)
	}, nats.Durable(consumer), nats.ManualAck())

	if err != nil {
		log.Error("nats: subscribe failed",
			"subject", subject,
			"consumer", consumer,
			"error", err,
		)
	}
}
