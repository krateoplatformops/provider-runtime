package workqueue

import (
	"math"
	"sync"
	"time"

	"k8s.io/client-go/util/workqueue"
)

type FailureRequest struct {
	// The number of times the request has been attempted
	Attempts int
	// The time at which the request was last attempted
	LastAttempt time.Time
}

type TypedItemExponentialTimedFailureRateLimiter[T comparable] struct {
	failuresLock sync.Mutex
	failures     map[T]FailureRequest

	baseDelay time.Duration
	maxDelay  time.Duration
}

func NewExponentialTimedFailureRateLimiter[T comparable](baseDelay time.Duration, maxDelay time.Duration) workqueue.TypedRateLimiter[T] {
	return &TypedItemExponentialTimedFailureRateLimiter[T]{
		failures:  map[T]FailureRequest{},
		baseDelay: baseDelay,
		maxDelay:  maxDelay,
	}
}

func (r *TypedItemExponentialTimedFailureRateLimiter[T]) When(item T) time.Duration {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	failreq, ok := r.failures[item]
	if !ok {
		r.failures[item] = FailureRequest{Attempts: 1, LastAttempt: time.Now()}
		return r.baseDelay
	}

	if time.Since(failreq.LastAttempt) > 2*r.maxDelay {
		return 0
	}

	exp := failreq.Attempts
	failreq.Attempts = failreq.Attempts + 1
	failreq.LastAttempt = time.Now()
	r.failures[item] = failreq

	// The backoff is capped such that 'calculated' value never overflows.
	backoff := float64(r.baseDelay.Nanoseconds()) * math.Pow(2, float64(exp))
	if backoff > math.MaxInt64 {
		return r.maxDelay
	}

	calculated := time.Duration(backoff)

	if calculated > r.maxDelay {
		return r.maxDelay
	}
	return calculated
}

func (r *TypedItemExponentialTimedFailureRateLimiter[T]) NumRequeues(item T) int {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	return r.failures[item].Attempts
}

func (r *TypedItemExponentialTimedFailureRateLimiter[T]) Forget(item T) {
	r.failuresLock.Lock()
	defer r.failuresLock.Unlock()

	if time.Since(r.failures[item].LastAttempt) > 2*r.maxDelay {
		delete(r.failures, item)
	}

}
