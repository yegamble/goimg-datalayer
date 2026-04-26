package community

import (
	"testing"
)

func TestGroupImageID(t *testing.T) {
	id1 := NewGroupImageID()
	id2 := NewGroupImageID()

	if id1.IsZero() {
		t.Error("NewGroupImageID should not be zero")
	}

	if id1.Equals(id2) {
		t.Error("Two new IDs should not be equal")
	}

	parsed, err := ParseGroupImageID(id1.String())
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if !id1.Equals(parsed) {
		t.Error("Parsed ID should equal original")
	}

	if id1.UUID() != parsed.UUID() {
		t.Error("UUIDs should match")
	}

	mustParsed := MustParseGroupImageID(id1.String())
	if !id1.Equals(mustParsed) {
		t.Error("MustParsed ID should equal original")
	}

	_, err = ParseGroupImageID("invalid")
	if err == nil {
		t.Error("Expected error parsing invalid ID")
	}

	zeroID := GroupImageID{}
	if !zeroID.IsZero() {
		t.Error("Empty struct should be zero")
	}
}

func TestMustParseGroupImageID_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseGroupImageID should panic on invalid string")
		}
	}()
	MustParseGroupImageID("invalid")
}
