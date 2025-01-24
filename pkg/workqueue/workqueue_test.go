package workqueue

import (
	"testing"
	"time"
)

func TestNewExponentialTimedFailureRateLimiter(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 10 * time.Second
	limiter := NewExponentialTimedFailureRateLimiter[string](baseDelay, maxDelay)

	if limiter == nil {
		t.Errorf("Expected non-nil limiter")
	}
}

func TestWhen(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 5 * time.Second
	limiter := NewExponentialTimedFailureRateLimiter[string](baseDelay, maxDelay)

	item := "testItem"
	delay := limiter.When(item)
	if delay != baseDelay {
		t.Errorf("Expected delay %v, got %v", baseDelay, delay)
	}

	delay = limiter.When(item)
	expectedDelay := 2 * baseDelay
	if delay != expectedDelay {
		t.Errorf("Expected delay %v, got %v", expectedDelay, delay)
	}

	delay = limiter.When(item)
	expectedDelay = 4 * baseDelay
	if delay != expectedDelay {
		t.Errorf("Expected delay %v, got %v", expectedDelay, delay)
	}

	delay = limiter.When(item)
	expectedDelay = maxDelay
	if delay != expectedDelay {
		t.Errorf("Expected delay %v, got %v", expectedDelay, delay)
	}

	// Test that the delay is reset after maxDelay
	time.Sleep(2 * maxDelay)
	delay = limiter.When(item)
	if delay != 0 {
		t.Errorf("Expected delay %v, got %v", baseDelay, delay)
	}
}

func TestNumRequeues(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 10 * time.Second
	limiter := NewExponentialTimedFailureRateLimiter[string](baseDelay, maxDelay)

	item := "testItem"
	limiter.When(item)
	limiter.When(item)

	numRequeues := limiter.NumRequeues(item)
	if numRequeues != 2 {
		t.Errorf("Expected numRequeues %v, got %v", 2, numRequeues)
	}
}

func TestForget(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 3 * time.Second
	limiter := NewExponentialTimedFailureRateLimiter[string](baseDelay, maxDelay)

	item := "testItem"
	limiter.When(item)

	time.Sleep(2 * maxDelay)

	limiter.Forget(item)

	numRequeues := limiter.NumRequeues(item)
	if numRequeues != 0 {
		t.Errorf("Expected numRequeues %v, got %v", 0, numRequeues)
	}
}
