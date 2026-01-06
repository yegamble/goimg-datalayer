package identity

import (
	"crypto/rand"
	"math/big"
	"time"
)

const (
	MinAuthDelay = 100 * time.Millisecond
	MaxAuthDelay = 300 * time.Millisecond
)

// CalculateRandomAuthDelay generates a cryptographically random delay
// between MinAuthDelay and MaxAuthDelay to mitigate timing attacks.
// If random generation fails, it returns MaxAuthDelay as a safe fallback.
func CalculateRandomAuthDelay() time.Duration {
	rangeMs := int64(MaxAuthDelay-MinAuthDelay) / int64(time.Millisecond)
	randomMs, err := rand.Int(rand.Reader, big.NewInt(rangeMs))
	if err != nil {
		return MaxAuthDelay
	}
	return MinAuthDelay + time.Duration(randomMs.Int64())*time.Millisecond
}

// ApplyAuthDelay sleeps for the remaining time to reach targetDelay.
// If actualProcessing already exceeded targetDelay, it sleeps for MinAuthDelay
// to maintain a minimum response time floor.
//
// Returns the actual delay applied (remaining or MinAuthDelay).
func ApplyAuthDelay(targetDelay, actualProcessing time.Duration) time.Duration {
	remaining := targetDelay - actualProcessing
	if remaining > 0 {
		time.Sleep(remaining)
		return remaining
	}
	time.Sleep(MinAuthDelay)
	return MinAuthDelay
}
