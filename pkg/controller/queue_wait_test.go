package controller

import (
	"context"
	"sync"
	"testing"
	"time"
)

type fakeQueueWaitRecorder struct {
	lock          sync.Mutex
	waitDurations []time.Duration
	depthDeltas   []int64
	oldestAges    []time.Duration
	workDurations []time.Duration
}

func (r *fakeQueueWaitRecorder) RecordQueueWaitDuration(_ context.Context, d time.Duration) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.waitDurations = append(r.waitDurations, d)
}

func (r *fakeQueueWaitRecorder) AddQueueDepth(_ context.Context, delta int64) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.depthDeltas = append(r.depthDeltas, delta)
}

func (r *fakeQueueWaitRecorder) RecordQueueOldestItemAge(_ context.Context, d time.Duration) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.oldestAges = append(r.oldestAges, d)
}

func (r *fakeQueueWaitRecorder) RecordQueueWorkDuration(_ context.Context, d time.Duration) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.workDurations = append(r.workDurations, d)
}

func TestQueueWaitInstrumentedRecordsQueueState(t *testing.T) {
	recorder := &fakeQueueWaitRecorder{}
	queue := &queueWaitInstrumented[string]{
		recorder:   recorder,
		enqueuedAt: map[string]time.Time{},
		startedAt:  map[string]time.Time{},
	}

	started := time.Unix(100, 0)
	inserted, oldestAge := queue.recordEnqueueAt("foo", started)
	if !inserted {
		t.Fatal("expected first enqueue to be recorded")
	}
	if oldestAge != 0 {
		t.Fatalf("expected oldest age after first enqueue = 0, got %v", oldestAge)
	}
	inserted, _ = queue.recordEnqueueAt("foo", started.Add(10*time.Second))
	if inserted {
		t.Fatal("expected duplicate enqueue to be ignored")
	}
	waitDuration, oldestAge, removed := queue.recordDequeueAt("foo", started.Add(25*time.Second))
	if !removed {
		t.Fatal("expected dequeue to remove pending item")
	}
	if waitDuration != 25*time.Second {
		t.Fatalf("recorded queue wait = %v, want %v", waitDuration, 25*time.Second)
	}
	if oldestAge != 0 {
		t.Fatalf("expected oldest age after dequeue = 0, got %v", oldestAge)
	}
	workDuration, ok := queue.recordDoneAt("foo", started.Add(40*time.Second))
	if !ok {
		t.Fatal("expected processing duration to be recorded")
	}
	if workDuration != 15*time.Second {
		t.Fatalf("recorded work duration = %v, want %v", workDuration, 15*time.Second)
	}

	if len(recorder.waitDurations) != 1 {
		t.Fatalf("expected 1 queue wait record, got %d", len(recorder.waitDurations))
	}
	if got, want := recorder.waitDurations[0], 25*time.Second; got != want {
		t.Fatalf("recorded queue wait = %v, want %v", got, want)
	}
	if got, want := recorder.depthDeltas, []int64{1, -1}; len(got) != len(want) {
		t.Fatalf("expected %d depth updates, got %d", len(want), len(got))
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("depth delta[%d] = %d, want %d", i, got[i], want[i])
			}
		}
	}
	if len(recorder.oldestAges) != 2 {
		t.Fatalf("expected 2 oldest-age samples, got %d", len(recorder.oldestAges))
	}
	if recorder.oldestAges[0] != 0 || recorder.oldestAges[1] != 0 {
		t.Fatalf("expected oldest ages to remain at zero, got %v", recorder.oldestAges)
	}
	if len(recorder.workDurations) != 1 {
		t.Fatalf("expected 1 work duration record, got %d", len(recorder.workDurations))
	}
	if got, want := recorder.workDurations[0], 15*time.Second; got != want {
		t.Fatalf("recorded work duration = %v, want %v", got, want)
	}
}

func TestQueueWaitInstrumentedIgnoresMissingEnqueue(t *testing.T) {
	recorder := &fakeQueueWaitRecorder{}
	queue := &queueWaitInstrumented[string]{
		recorder:   recorder,
		enqueuedAt: map[string]time.Time{},
		startedAt:  map[string]time.Time{},
	}

	_, _, removed := queue.recordDequeueAt("foo", time.Unix(100, 0))
	if removed {
		t.Fatal("expected missing enqueue to be ignored")
	}
	if workDuration, ok := queue.recordDoneAt("foo", time.Unix(125, 0)); ok || workDuration != 0 {
		t.Fatalf("expected missing processing entry to be ignored, got duration=%v ok=%v", workDuration, ok)
	}

	if len(recorder.waitDurations) != 0 {
		t.Fatalf("expected no queue wait records, got %d", len(recorder.waitDurations))
	}
	if len(recorder.depthDeltas) != 0 {
		t.Fatalf("expected no depth updates, got %d", len(recorder.depthDeltas))
	}
	if len(recorder.oldestAges) != 0 {
		t.Fatalf("expected no oldest-age samples, got %d", len(recorder.oldestAges))
	}
	if len(recorder.workDurations) != 0 {
		t.Fatalf("expected no work duration records, got %d", len(recorder.workDurations))
	}
}

func TestOptionsForControllerRuntimeQueueWaitRecorder(t *testing.T) {
	tests := map[string]struct {
		opts         Options
		wantNewQueue bool
	}{
		"recorder disabled": {
			opts:         Options{},
			wantNewQueue: false,
		},
		"recorder enabled": {
			opts: Options{
				QueueWaitRecorder: &fakeQueueWaitRecorder{},
			},
			wantNewQueue: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			runtimeOpts := tc.opts.ForControllerRuntime()
			if tc.wantNewQueue && runtimeOpts.NewQueue == nil {
				t.Fatal("expected custom NewQueue to be configured")
			}
			if !tc.wantNewQueue && runtimeOpts.NewQueue != nil {
				t.Fatal("did not expect a custom NewQueue to be configured")
			}
		})
	}
}
