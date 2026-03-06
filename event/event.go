package event

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// BaseEvent is the standard envelope for every domain event published to NATS JetStream.
//
// Subject format: tais.{service}.{entity}.{event}
// Stream:         TAIS_EVENTS
//
// Embed BaseEvent in every service-level event struct:
//
//	type VehicleRegisteredEvent struct {
//	    pkgevent.BaseEvent
//	    VehicleID uint   `json:"vehicle_id"`
//	    VIN       string `json:"vin"`
//	}
//
//	subj := pkgevent.Subject("registration", "vehicle", "registered")
//	ev := VehicleRegisteredEvent{
//	    BaseEvent: pkgevent.New(subj, "tais-registration", &actorID, payload),
//	    VehicleID: v.ID,
//	    VIN:       v.VIN,
//	}
type BaseEvent struct {
	// ID is a UUID v4 — unique event identifier used for deduplication in tais-audit.
	ID string `json:"id"`

	// Subject is the NATS subject this event was published on.
	// e.g. "tais.vehicle.vehicle.created"
	Subject string `json:"subject"`

	// Service identifies the publishing service name (e.g. "tais-vehicle").
	Service string `json:"service"`

	// ActorID is the staff user ID who triggered the event.
	// Nil for system-generated events (migrations, scheduled jobs).
	ActorID *uint `json:"actor_id"`

	// OccurredAt is the UTC timestamp when the event was created.
	OccurredAt time.Time `json:"occurred_at"`

	// Payload holds the event-specific data.
	Payload any `json:"payload"`
}

// New creates a BaseEvent with all metadata populated.
// Pass nil for actorID on system-generated events.
//
//	subj := pkgevent.Subject("vehicle", "vehicle", "created")
//	base := pkgevent.New(subj, "tais-vehicle", &actorID, payload)
func New(subject, service string, actorID *uint, payload any) BaseEvent {
	return BaseEvent{
		ID:         uuid.New().String(),
		Subject:    subject,
		Service:    service,
		ActorID:    actorID,
		OccurredAt: time.Now().UTC(),
		Payload:    payload,
	}
}

// Subject builds a NATS subject following the TAIS naming convention.
//
//	Subject("vehicle", "vehicle", "created") → "tais.vehicle.vehicle.created"
func Subject(service, entity, eventType string) string {
	return fmt.Sprintf("tais.%s.%s.%s", service, entity, eventType)
}
