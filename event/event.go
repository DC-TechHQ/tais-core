package event

import (
	"fmt"
	"time"
)

// BaseEvent is embedded in all domain events published to NATS JetStream.
// Subject format: tais.{service}.{entity}.{event}
// Stream: TAIS_EVENTS
type BaseEvent struct {
	// ID is a unique event identifier (UUID). Used for deduplication.
	ID string `json:"id"`

	// Subject is the NATS subject this event was published on.
	// e.g. "tais.vehicle.vehicle.created"
	Subject string `json:"subject"`

	// OccurredAt is the UTC timestamp when the event was created.
	OccurredAt time.Time `json:"occurred_at"`

	// ActorID is the user ID who triggered the event (0 for system events).
	ActorID uint `json:"actor_id,omitempty"`

	// ServiceName identifies the publishing service (e.g. "tais-vehicle").
	ServiceName string `json:"service_name"`
}

// New creates a BaseEvent with subject and metadata populated.
// payload is set by each concrete event type.
//
//	ev := event.New("tais-vehicle", "vehicle", "vehicle", "created", actorID)
func New(serviceName, service, entity, eventType string, actorID uint) BaseEvent {
	return BaseEvent{
		ID:          newID(),
		Subject:     Subject(service, entity, eventType),
		OccurredAt:  time.Now().UTC(),
		ActorID:     actorID,
		ServiceName: serviceName,
	}
}

// Subject builds a NATS subject following the tais naming convention.
//
//	Subject("vehicle", "vehicle", "created") → "tais.vehicle.vehicle.created"
func Subject(service, entity, eventType string) string {
	return fmt.Sprintf("tais.%s.%s.%s", service, entity, eventType)
}

// newID generates a simple unique ID using timestamp + random suffix.
// In production services, replace with github.com/google/uuid if available.
func newID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
