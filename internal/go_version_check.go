//go:build go1.25

// Package internal contains shared internal utilities.
// This file enforces Go 1.25+ at compile time using build constraints.
//
// If you see a build error mentioning this file, you need to upgrade Go:
//   - Required: Go 1.25 or later
//   - Download: https://go.dev/dl/
//   - Current stable: Go 1.25.5
//
// The go.mod file also specifies `go 1.25` and `toolchain go1.25.5`,
// but this build constraint provides an additional compile-time check.
package internal
