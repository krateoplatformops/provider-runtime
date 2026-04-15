package controller

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/util/workqueue"
	sigspq "sigs.k8s.io/controller-runtime/pkg/controller/priorityqueue"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// QueueWaitRecorder records queue telemetry for a controller queue.
type QueueWaitRecorder interface {
	RecordQueueWaitDuration(ctx context.Context, d time.Duration)
	AddQueueDepth(ctx context.Context, delta int64)
	RecordQueueOldestItemAge(ctx context.Context, d time.Duration)
	RecordQueueWorkDuration(ctx context.Context, d time.Duration)
}

func newQueueWithWaitMetrics(controllerName string, rateLimiter workqueue.TypedRateLimiter[reconcile.Request], usePriorityQueue bool, recorder QueueWaitRecorder) workqueue.TypedRateLimitingInterface[reconcile.Request] {
	if recorder == nil {
		return newControllerQueue(controllerName, rateLimiter, usePriorityQueue)
	}

	return &queueWaitInstrumented[reconcile.Request]{
		queue:      newControllerQueue(controllerName, rateLimiter, usePriorityQueue),
		recorder:   recorder,
		enqueuedAt: make(map[reconcile.Request]time.Time),
		startedAt:  make(map[reconcile.Request]time.Time),
	}
}

func newControllerQueue(controllerName string, rateLimiter workqueue.TypedRateLimiter[reconcile.Request], usePriorityQueue bool) workqueue.TypedRateLimitingInterface[reconcile.Request] {
	if usePriorityQueue {
		return sigspq.New(controllerName, func(o *sigspq.Opts[reconcile.Request]) {
			o.Log = logr.Discard()
			o.RateLimiter = rateLimiter
		})
	}

	return workqueue.NewTypedRateLimitingQueueWithConfig(rateLimiter, workqueue.TypedRateLimitingQueueConfig[reconcile.Request]{
		Name: controllerName,
	})
}

type queueWaitInstrumented[T comparable] struct {
	queue      workqueue.TypedRateLimitingInterface[T]
	recorder   QueueWaitRecorder
	lock       sync.Mutex
	enqueuedAt map[T]time.Time
	startedAt  map[T]time.Time
}

func (q *queueWaitInstrumented[T]) Add(item T) {
	q.recordEnqueueAt(item, time.Now())
	q.queue.Add(item)
}

func (q *queueWaitInstrumented[T]) AddAfter(item T, duration time.Duration) {
	q.recordEnqueueAt(item, time.Now())
	q.queue.AddAfter(item, duration)
}

func (q *queueWaitInstrumented[T]) AddRateLimited(item T) {
	q.recordEnqueueAt(item, time.Now())
	q.queue.AddRateLimited(item)
}

func (q *queueWaitInstrumented[T]) Get() (item T, shutdown bool) {
	item, shutdown = q.queue.Get()
	if !shutdown {
		q.recordDequeueAt(item, time.Now())
	}
	return item, shutdown
}

func (q *queueWaitInstrumented[T]) Done(item T) {
	q.recordDoneAt(item, time.Now())
	q.queue.Done(item)
}

func (q *queueWaitInstrumented[T]) ShutDown() {
	if pending := q.resetState(); pending > 0 {
		q.recorder.AddQueueDepth(context.Background(), -pending)
	}
	q.queue.ShutDown()
}

func (q *queueWaitInstrumented[T]) ShutDownWithDrain() {
	q.queue.ShutDownWithDrain()
}

func (q *queueWaitInstrumented[T]) ShuttingDown() bool {
	return q.queue.ShuttingDown()
}

func (q *queueWaitInstrumented[T]) Len() int {
	return q.queue.Len()
}

func (q *queueWaitInstrumented[T]) Forget(item T) {
	q.queue.Forget(item)
}

func (q *queueWaitInstrumented[T]) NumRequeues(item T) int {
	return q.queue.NumRequeues(item)
}

func (q *queueWaitInstrumented[T]) recordEnqueueAt(item T, started time.Time) (bool, time.Duration) {
	q.lock.Lock()

	if _, exists := q.enqueuedAt[item]; exists {
		oldestAge := q.oldestItemAgeLocked(started)
		q.lock.Unlock()
		return false, oldestAge
	}

	q.enqueuedAt[item] = started
	oldestAge := q.oldestItemAgeLocked(started)
	q.lock.Unlock()

	q.recorder.AddQueueDepth(context.Background(), 1)
	q.recorder.RecordQueueOldestItemAge(context.Background(), oldestAge)

	return true, oldestAge
}

func (q *queueWaitInstrumented[T]) recordDequeue(item T) (time.Duration, time.Duration, bool) {
	return q.recordDequeueAt(item, time.Now())
}

func (q *queueWaitInstrumented[T]) recordDequeueAt(item T, finished time.Time) (time.Duration, time.Duration, bool) {
	q.lock.Lock()
	started, exists := q.enqueuedAt[item]
	if exists {
		delete(q.enqueuedAt, item)
		q.startedAt[item] = finished
	}
	oldestAge := q.oldestItemAgeLocked(finished)
	q.lock.Unlock()

	if !exists {
		return 0, oldestAge, false
	}

	q.recorder.AddQueueDepth(context.Background(), -1)
	q.recorder.RecordQueueWaitDuration(context.Background(), finished.Sub(started))
	q.recorder.RecordQueueOldestItemAge(context.Background(), oldestAge)

	return finished.Sub(started), oldestAge, true
}

func (q *queueWaitInstrumented[T]) recordDoneAt(item T, finished time.Time) (time.Duration, bool) {
	q.lock.Lock()
	started, exists := q.startedAt[item]
	if exists {
		delete(q.startedAt, item)
	}
	q.lock.Unlock()

	if !exists {
		return 0, false
	}

	workDuration := finished.Sub(started)
	q.recorder.RecordQueueWorkDuration(context.Background(), workDuration)

	return workDuration, true
}

func (q *queueWaitInstrumented[T]) resetState() int64 {
	q.lock.Lock()
	pending := int64(len(q.enqueuedAt))
	q.enqueuedAt = make(map[T]time.Time)
	q.startedAt = make(map[T]time.Time)
	q.lock.Unlock()

	return pending
}

func (q *queueWaitInstrumented[T]) oldestItemAgeLocked(now time.Time) time.Duration {
	var oldest time.Duration
	for _, started := range q.enqueuedAt {
		age := now.Sub(started)
		if age > oldest {
			oldest = age
		}
	}

	return oldest
}
