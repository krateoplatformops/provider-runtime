package controller

import (
	"time"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"github.com/krateoplatformops/provider-runtime/pkg/ratelimiter"
)

// DefaultOptions returns a functional set of options with conservative
// defaults.
func DefaultOptions() Options {
	return Options{
		Logger:                  logging.NewNopLogger(),
		GlobalRateLimiter:       ratelimiter.NewGlobal(1),
		PollInterval:            1 * time.Minute,
		MaxConcurrentReconciles: 1,
	}
}

// Options frequently used by most controllers.
type Options struct {
	// The Logger controllers should use.
	Logger logging.Logger

	// The GlobalRateLimiter used by this controller manager. The rate of
	// reconciles across all controllers will be subject to this limit.
	GlobalRateLimiter workqueue.RateLimiter

	// PollInterval at which each controller should speculatively poll to
	// determine whether it has work to do.
	PollInterval time.Duration

	// MaxConcurrentReconciles for each controller.
	MaxConcurrentReconciles int
}

// ForControllerRuntime extracts options for controller-runtime.
func (o Options) ForControllerRuntime() controller.Options {
	return controller.Options{
		MaxConcurrentReconciles: o.MaxConcurrentReconciles,
		RateLimiter:             ratelimiter.NewController(),
	}
}
