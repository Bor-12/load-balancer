package app

import "testing"

func TestStartupMessage(t *testing.T) {
	expectedMessage := "Load Balancer starting"

	if StartupMessage() != expectedMessage {
		t.Fatalf("expected %q, got %q", expectedMessage, StartupMessage())
	}
}
