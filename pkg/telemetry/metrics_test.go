package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"go.opentelemetry.io/otel/sdk/metric"
	metricdata "go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestSetupDisabledReturnsNoopShutdown(t *testing.T) {
	metrics, shutdown, err := Setup(context.Background(), logging.NewNopLogger(), Config{})
	if err != nil {
		t.Fatalf("Setup() returned error: %v", err)
	}
	if metrics != nil {
		t.Fatalf("Setup() metrics = %#v, want nil", metrics)
	}
	if shutdown == nil {
		t.Fatal("Setup() shutdown = nil, want no-op shutdown")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() returned error: %v", err)
	}
}

func TestNewMetricsRecordsData(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	ctx := context.Background()
	t.Cleanup(func() {
		if err := provider.Shutdown(ctx); err != nil {
			t.Fatalf("provider.Shutdown() returned error: %v", err)
		}
	})

	metrics, err := newMetrics(provider.Meter("github.com/krateoplatformops/provider-runtime/test"))
	if err != nil {
		t.Fatalf("newMetrics() returned error: %v", err)
	}

	metrics.IncStartupSuccess(ctx)
	metrics.RecordGetDuration(ctx, 25*time.Millisecond)
	metrics.IncGetFailure(ctx)
	metrics.RecordReconcileDuration(ctx, 150*time.Millisecond)
	metrics.AddQueueDepth(ctx, 1)
	metrics.RecordQueueWaitDuration(ctx, 75*time.Millisecond)
	metrics.RecordQueueOldestItemAge(ctx, 5*time.Second)
	metrics.RecordQueueWorkDuration(ctx, 125*time.Millisecond)
	metrics.IncReconcileRequeueAfter(ctx)
	metrics.IncReconcileRequeueImmediate(ctx)
	metrics.IncReconcileErrorRequeue(ctx)
	metrics.AddReconcileInFlight(1)
	metrics.IncReconcileFailure(ctx)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("reader.Collect() returned error: %v", err)
	}

	if !hasMetric(rm, "provider_runtime.startup.success") {
		t.Fatal("expected startup success metric to be collected")
	}
	if !hasMetric(rm, "provider_runtime.reconcile.get.duration_seconds") {
		t.Fatal("expected get duration metric to be collected")
	}
	if !hasMetric(rm, "provider_runtime.reconcile.get.failure") {
		t.Fatal("expected get failure metric to be collected")
	}
	if !hasMetric(rm, "provider_runtime.reconcile.duration_seconds") {
		t.Fatal("expected reconcile duration metric to be collected")
	}
	if !hasMetric(rm, "provider_runtime.reconcile.queue.depth") {
		t.Fatal("expected queue depth metric to be collected")
	}
	if !hasMetric(rm, "provider_runtime.reconcile.queue.wait.duration_seconds") {
		t.Fatal("expected queue wait metric to be collected")
	}
	if !hasMetric(rm, "provider_runtime.reconcile.queue.oldest_item_age_seconds") {
		t.Fatal("expected queue oldest item age metric to be collected")
	}
	if !hasMetric(rm, "provider_runtime.reconcile.queue.work.duration_seconds") {
		t.Fatal("expected queue work duration metric to be collected")
	}
	if !hasMetric(rm, "provider_runtime.reconcile.queue.requeues") {
		t.Fatal("expected queue requeue total metric to be collected")
	}
	if !hasMetric(rm, "provider_runtime.reconcile.requeue.after") {
		t.Fatal("expected reconcile requeue-after metric to be collected")
	}
	if !hasMetric(rm, "provider_runtime.reconcile.requeue.immediate") {
		t.Fatal("expected reconcile requeue-immediate metric to be collected")
	}
	if !hasMetric(rm, "provider_runtime.reconcile.requeue.error") {
		t.Fatal("expected reconcile requeue-error metric to be collected")
	}
	if !hasMetric(rm, "provider_runtime.reconcile.in_flight") {
		t.Fatal("expected reconcile in-flight metric to be collected")
	}
	if !hasMetric(rm, "provider_runtime.reconcile.failure") {
		t.Fatal("expected reconcile failure metric to be collected")
	}
}

func hasMetric(rm metricdata.ResourceMetrics, name string) bool {
	for _, scope := range rm.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name == name {
				return true
			}
		}
	}

	return false
}
