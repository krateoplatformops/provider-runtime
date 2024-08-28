// Package ratelimiter contains suggested default ratelimiters for providers.
package ratelimiter

import (
	"time"

	"golang.org/x/time/rate"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/workqueue"
)

// NewGlobal returns a token bucket rate limiter meant for limiting the number
// of average total requeues per second for all controllers registered with a
// controller manager. The bucket size (i.e. allowed burst) is rps * 10.
func NewGlobal(rps int) *workqueue.TypedBucketRateLimiter[any] {
	return &workqueue.TypedBucketRateLimiter[any]{Limiter: rate.NewLimiter(rate.Limit(rps), rps*10)}
}

// NewController returns a rate limiter that takes the maximum delay between the
// passed rate limiter and a per-item exponential backoff limiter. The
// exponential backoff limiter has a base delay of 1s and a maximum of 60s.
func NewController() workqueue.TypedRateLimiter[any] {
	return workqueue.NewTypedItemExponentialFailureRateLimiter[any](1*time.Second, 60*time.Second)
}

// LimitRESTConfig returns a copy of the supplied REST config with rate limits
// derived from the supplied rate of reconciles per second.
func LimitRESTConfig(cfg *rest.Config, rps int) *rest.Config {
	// The Kubernetes controller manager and controller-runtime controller
	// managers use 20qps with 30 burst. We default to 10 reconciles per
	// second so our defaults are designed to accommodate that.
	out := rest.CopyConfig(cfg)
	out.QPS = float32(rps * 5)
	out.Burst = rps * 10
	return out
}
