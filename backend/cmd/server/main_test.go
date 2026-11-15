package main

import "testing"

// TestRequestTimeoutBelowWriteTimeout guards D-21: the request deadline
// must pass before the server WriteTimeout, or the JSON error response
// could not be written before the connection is cut.
func TestRequestTimeoutBelowWriteTimeout(t *testing.T) {
	t.Parallel()

	if requestTimeout <= 0 || requestTimeout >= writeTimeout {
		t.Fatalf("requestTimeout = %v, want 0 < requestTimeout < writeTimeout (%v)", requestTimeout, writeTimeout)
	}
}
