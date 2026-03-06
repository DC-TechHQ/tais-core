package event_test

import (
	"strings"
	"testing"

	"github.com/DC-TechHQ/tais-core/event"
)

func TestSubject(t *testing.T) {
	got := event.Subject("vehicle", "vehicle", "created")
	want := "tais.vehicle.vehicle.created"
	if got != want {
		t.Errorf("Subject: got %q, want %q", got, want)
	}
}

func TestSubject_Format(t *testing.T) {
	s := event.Subject("registration", "owner", "transferred")
	if !strings.HasPrefix(s, "tais.") {
		t.Error("Subject must start with 'tais.'")
	}
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		t.Errorf("Subject must have 4 dot-separated parts, got %d: %q", len(parts), s)
	}
}

func TestNew_WithActor(t *testing.T) {
	actorID := uint(42)
	subj := event.Subject("vehicle", "vehicle", "created")
	payload := map[string]any{"vin": "WVWZZZ1KZ"}

	ev := event.New(subj, "tais-vehicle", &actorID, payload)

	if ev.ID == "" {
		t.Error("expected non-empty ID")
	}
	if ev.Subject != subj {
		t.Errorf("Subject: got %q, want %q", ev.Subject, subj)
	}
	if ev.Service != "tais-vehicle" {
		t.Errorf("Service: got %q, want tais-vehicle", ev.Service)
	}
	if ev.ActorID == nil || *ev.ActorID != 42 {
		t.Error("ActorID should be 42")
	}
	if ev.OccurredAt.IsZero() {
		t.Error("OccurredAt should not be zero")
	}
	if ev.Payload == nil {
		t.Error("Payload should not be nil")
	}
}

func TestNew_SystemEvent_NilActor(t *testing.T) {
	subj := event.Subject("audit", "record", "imported")
	ev := event.New(subj, "tais-audit", nil, nil)

	if ev.ActorID != nil {
		t.Error("system event should have nil ActorID")
	}
}

func TestNew_UniqueIDs(t *testing.T) {
	subj := event.Subject("vehicle", "vehicle", "created")
	ev1 := event.New(subj, "tais-vehicle", nil, nil)
	ev2 := event.New(subj, "tais-vehicle", nil, nil)

	if ev1.ID == ev2.ID {
		t.Error("consecutive events must have different IDs")
	}
}
