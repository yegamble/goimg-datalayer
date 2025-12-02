// Package helpers provides common test utilities for the goimg-datalayer project.
package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RequireNoError fails the test immediately if err is not nil.
func RequireNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	require.NoError(t, err, msgAndArgs...)
}

// RequireError fails the test immediately if err is nil.
func RequireError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	require.Error(t, err, msgAndArgs...)
}

// AssertEqual checks if expected equals actual.
func AssertEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Equal(t, expected, actual, msgAndArgs...)
}

// AssertTrue checks if condition is true.
func AssertTrue(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	assert.True(t, condition, msgAndArgs...)
}

// AssertFalse checks if condition is false.
func AssertFalse(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	assert.False(t, condition, msgAndArgs...)
}

// AssertNil checks if object is nil.
func AssertNil(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Nil(t, object, msgAndArgs...)
}

// AssertNotNil checks if object is not nil.
func AssertNotNil(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.NotNil(t, object, msgAndArgs...)
}
