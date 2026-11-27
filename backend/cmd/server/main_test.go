package main

import "testing"

func TestRequestTimeoutBelowWriteTimeout(t *testing.T) {
	t.Parallel()

	if requestTimeout <= 0 || requestTimeout >= writeTimeout {
		t.Fatalf("requestTimeout = %v, want 0 < requestTimeout < writeTimeout (%v)", requestTimeout, writeTimeout)
	}
}
