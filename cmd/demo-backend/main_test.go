package main

import (
	"testing"
	"time"
)

func TestResponseDuration_ReturnsBaseDelayWithoutJitter(t *testing.T) {
	baseDelay := 1500 * time.Millisecond

	delay := responseDuration(baseDelay, 0)

	if delay != baseDelay {
		t.Fatalf("expected delay %s, got %s", baseDelay, delay)
	}
}

func TestResponseDuration_AddsJitterWithinRange(t *testing.T) {
	baseDelay := 500 * time.Millisecond
	jitter := 2 * time.Second

	for range 100 {
		delay := responseDuration(baseDelay, jitter)
		if delay < baseDelay {
			t.Fatalf("expected delay to be at least %s, got %s", baseDelay, delay)
		}
		if delay > baseDelay+jitter {
			t.Fatalf("expected delay to be at most %s, got %s", baseDelay+jitter, delay)
		}
	}
}
