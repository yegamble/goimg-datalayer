package community

import (
	"testing"
)

func TestGroupActivityID(t *testing.T) {
	id1 := NewGroupActivityID()
	id2 := NewGroupActivityID()

	if id1.Equals(id2) {
		t.Error("Two new IDs should not be equal")
	}

	parsed, err := ParseGroupActivityID(id1.String())
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if !id1.Equals(parsed) {
		t.Error("Parsed ID should equal original")
	}

	mustParsed := MustParseGroupActivityID(id1.String())
	if !id1.Equals(mustParsed) {
		t.Error("MustParsed ID should equal original")
	}

	_, err = ParseGroupActivityID("invalid")
	if err == nil {
		t.Error("Expected error parsing invalid ID")
	}
}

func TestMustParseGroupActivityID_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseGroupActivityID should panic on invalid string")
		}
	}()
	MustParseGroupActivityID("invalid")
}
