package helpers_test

import (
	"testing"

	"github.com/yegamble/goimg-datalayer/tests/helpers"
)

// TestHelperFunctions verifies that test helper utilities work correctly.
func TestHelperFunctions(t *testing.T) {
	t.Parallel()

	t.Run("TestUserID returns consistent UUID", func(t *testing.T) {
		t.Parallel()

		id1 := helpers.TestUserID()
		id2 := helpers.TestUserID()

		helpers.AssertEqual(t, id1, id2, "TestUserID should return consistent UUID")
		helpers.AssertEqual(t, "00000000-0000-0000-0000-000000000001", id1.String())
	})

	t.Run("TestEmail returns consistent email", func(t *testing.T) {
		t.Parallel()

		email := helpers.TestEmail()
		helpers.AssertEqual(t, "test@example.com", email)
	})

	t.Run("TestUsername returns consistent username", func(t *testing.T) {
		t.Parallel()

		username := helpers.TestUsername()
		helpers.AssertEqual(t, "testuser", username)
	})

	t.Run("RandomUUID generates unique UUIDs", func(t *testing.T) {
		t.Parallel()

		id1 := helpers.RandomUUID()
		id2 := helpers.RandomUUID()

		helpers.AssertNotNil(t, id1)
		helpers.AssertNotNil(t, id2)
		helpers.AssertFalse(t, id1 == id2, "Random UUIDs should be different")
	})
}

// TestAssertHelpers verifies assertion helper functions.
func TestAssertHelpers(t *testing.T) {
	t.Parallel()

	t.Run("AssertTrue passes on true condition", func(t *testing.T) {
		t.Parallel()
		helpers.AssertTrue(t, true)
	})

	t.Run("AssertFalse passes on false condition", func(t *testing.T) {
		t.Parallel()
		helpers.AssertFalse(t, false)
	})

	t.Run("AssertEqual compares values", func(t *testing.T) {
		t.Parallel()
		helpers.AssertEqual(t, 42, 42)
		helpers.AssertEqual(t, "test", "test")
	})

	t.Run("AssertNil checks nil values", func(t *testing.T) {
		t.Parallel()
		var nilValue *string
		helpers.AssertNil(t, nilValue)
	})

	t.Run("AssertNotNil checks non-nil values", func(t *testing.T) {
		t.Parallel()
		value := "not nil"
		helpers.AssertNotNil(t, &value)
	})
}

// TestRequireHelpers verifies require helper functions.
func TestRequireHelpers(t *testing.T) {
	t.Parallel()

	t.Run("RequireNoError passes on nil error", func(t *testing.T) {
		t.Parallel()
		helpers.RequireNoError(t, nil)
	})

	t.Run("RequireError passes on non-nil error", func(t *testing.T) {
		t.Parallel()
		err := &testError{msg: "test error"}
		helpers.RequireError(t, err)
	})
}

// testError is a simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
