package ratelimiter

import (
	"testing"
	"time"
)

func TestDefaultManagedRateLimiter(t *testing.T) {
	limiter := NewController()
	backoffSchedule := []int{1, 2, 4, 8, 16, 32, 60}
	for _, d := range backoffSchedule {
		if e, a := time.Duration(d)*time.Second, limiter.When("one"); e != a {
			t.Errorf("expected %v, got %v", e, a)
		}
	}
	limiter.Forget("one")
	if e, a := time.Duration(backoffSchedule[0])*time.Second, limiter.When("one"); e != a {
		t.Errorf("expected %v, got %v", e, a)
	}
}
