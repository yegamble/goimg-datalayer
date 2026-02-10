package containers

import (
	"fmt"
	"strings"
	"testing"
)

var dockerUnavailableMarkers = []string{
	"rootless docker not found",
	"cannot connect to the docker daemon",
	"is the docker daemon running",
	"docker daemon",
	"docker socket",
	"docker host",
	"no such file or directory",
	"permission denied",
}

func isDockerUnavailable(err error) bool {
	if err == nil {
		return false
	}
	return hasDockerUnavailableMarker(err.Error())
}

func isDockerUnavailablePanic(v interface{}) bool {
	if v == nil {
		return false
	}
	return hasDockerUnavailableMarker(fmt.Sprint(v))
}

func hasDockerUnavailableMarker(message string) bool {
	lower := strings.ToLower(message)
	for _, marker := range dockerUnavailableMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func skipDockerUnavailable(t testing.TB, reason interface{}) {
	if t == nil {
		return
	}
	t.Helper()
	t.Skipf("skipping integration test because Docker is unavailable: %v", reason)
}
