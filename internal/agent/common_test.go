// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/go-openapi/strfmt"
	"github.com/pashagolub/pgxmock/v5"

	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/internal/db"
)

// fakeWorker's ProcessServices fails failuresLeft times before succeeding.
// When busy is true, those failures are ErrProcessBusy (benign serialization)
// rather than a generic error. When gate is non-nil, ProcessServices blocks on
// it, letting a test hold a run "in progress" while firing more enqueues.
type fakeWorker struct {
	scheduler    gocron.Scheduler
	failuresLeft atomic.Int32
	calls        atomic.Int32
	busy         bool
	gate         chan struct{}
	coalescer    *Coalescer
}

func (w *fakeWorker) ProcessServices(context.Context) error {
	w.calls.Add(1)
	if w.gate != nil {
		<-w.gate
	}
	if w.failuresLeft.Add(-1) >= 0 {
		if w.busy {
			return ErrProcessBusy
		}
		return errors.New("boom")
	}
	return nil
}
func (w *fakeWorker) ProcessEndpoint(context.Context, strfmt.UUID) error { return nil }
func (w *fakeWorker) GetPool() db.PgxIface                               { return nil }
func (w *fakeWorker) GetScheduler() gocron.Scheduler                     { return w.scheduler }

// ProcessServicesCoalescer satisfies the optional interface ScheduleProcessServices checks.
func (w *fakeWorker) ProcessServicesCoalescer() *Coalescer { return w.coalescer }

// TestScheduleProcessServicesRetriesUntilSuccess verifies a failing ProcessServices run is retried until it succeeds.
func TestScheduleProcessServicesRetriesUntilSuccess(t *testing.T) {
	// Shrink the retry delay for the test.
	origDelay := processServicesRetryDelay
	processServicesRetryDelay = 5 * time.Millisecond
	defer func() { processServicesRetryDelay = origDelay }()

	scheduler, err := gocron.NewScheduler(gocron.WithLimitConcurrentJobs(1, gocron.LimitModeWait))
	if err != nil {
		t.Fatal(err)
	}
	scheduler.Start()
	defer func() { _ = scheduler.Shutdown() }()

	w := &fakeWorker{scheduler: scheduler}
	w.failuresLeft.Store(2) // fail twice, then succeed on the third call

	if _, err := scheduler.NewJob(
		gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
		gocron.NewTask(processServicesTask(context.Background(), scheduler, w)),
		gocron.WithName("ProcessServices"),
	); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(2 * time.Second)
	for w.calls.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("ProcessServices was called %d times, expected 3 (2 failures + 1 success)", w.calls.Load())
		case <-time.After(2 * time.Millisecond):
		}
	}
	// Give a moment to ensure it does not keep retrying after success.
	time.Sleep(30 * time.Millisecond)
	if got := w.calls.Load(); got != 3 {
		t.Errorf("ProcessServices called %d times, expected exactly 3 (no retry after success)", got)
	}
}

// TestScheduleProcessServicesBusyReschedules verifies that an ErrProcessBusy
// result reschedules promptly (so a NOTIFY-driven run that loses the advisory
// lock is re-driven without waiting for PendingSyncLoop) and never surfaces as
// a job failure.
func TestScheduleProcessServicesBusyReschedules(t *testing.T) {
	origBusy := processServicesBusyDelay
	processServicesBusyDelay = 5 * time.Millisecond
	defer func() { processServicesBusyDelay = origBusy }()

	scheduler, err := gocron.NewScheduler(gocron.WithLimitConcurrentJobs(1, gocron.LimitModeWait))
	if err != nil {
		t.Fatal(err)
	}
	scheduler.Start()
	defer func() { _ = scheduler.Shutdown() }()

	w := &fakeWorker{scheduler: scheduler, busy: true}
	w.failuresLeft.Store(1) // busy once, then succeed

	run := processServicesTask(context.Background(), scheduler, w)
	if err := run(); err != nil {
		t.Errorf("processServicesTask returned %v, expected nil for ErrProcessBusy", err)
	}

	// The busy result must trigger a reschedule that then succeeds.
	deadline := time.After(2 * time.Second)
	for w.calls.Load() < 2 {
		select {
		case <-deadline:
			t.Fatalf("ProcessServices called %d times, expected reschedule after busy", w.calls.Load())
		case <-time.After(2 * time.Millisecond):
		}
	}
}

func TestRegisterAgent(t *testing.T) {
	config.Global.Default.Host = "test-host"
	config.Global.Default.AvailabilityZone = ""
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	var nilString *string
	dbMock.
		ExpectExec("INSERT INTO agents (host,availability_zone,provider,physnet) VALUES ($1,$2,$3,$4) ON CONFLICT (host) DO UPDATE SET availability_zone = $5, physnet = $6, updated_at = now(), heartbeat_at = now(), enabled = true").
		WithArgs(config.Global.Default.Host, nilString, "test", nilString, nilString, nilString).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	RegisterAgent(dbMock, "test")
}

func TestRegisterAgentWithAZ(t *testing.T) {
	config.Global.Default.Host = "test-host"
	config.Global.Default.AvailabilityZone = "test-az"
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	var nilString *string
	dbMock.
		ExpectExec("INSERT INTO agents (host,availability_zone,provider,physnet) VALUES ($1,$2,$3,$4) ON CONFLICT (host) DO UPDATE SET availability_zone = $5, physnet = $6, updated_at = now(), heartbeat_at = now(), enabled = true").
		WithArgs(config.Global.Default.Host, &config.Global.Default.AvailabilityZone, "test", nilString, &config.Global.Default.AvailabilityZone, nilString).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	RegisterAgent(dbMock, "test")
}

func TestRegisterAgentWith(t *testing.T) {
	config.Global.Default.Host = "test-host"
	config.Global.Default.AvailabilityZone = "test-az"
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	var nilString *string
	dbMock.
		ExpectExec("INSERT INTO agents (host,availability_zone,provider,physnet) VALUES ($1,$2,$3,$4) ON CONFLICT (host) DO UPDATE SET availability_zone = $5, physnet = $6, updated_at = now(), heartbeat_at = now(), enabled = true").
		WithArgs(config.Global.Default.Host, &config.Global.Default.AvailabilityZone, "test", nilString, &config.Global.Default.AvailabilityZone, nilString).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	RegisterAgent(dbMock, "test")
}

// TestCoalescer_StateMachine verifies the edge-triggered dedup semantics directly:
// first request enqueues, further requests during a run coalesce into exactly one rerun.
func TestCoalescer_StateMachine(t *testing.T) {
	var c Coalescer

	// First request wins and should enqueue.
	if !c.requestEnqueue() {
		t.Fatal("first requestEnqueue should return true (enqueue)")
	}
	// Subsequent requests while running must coalesce (return false).
	for i := 0; i < 5; i++ {
		if c.requestEnqueue() {
			t.Fatalf("requestEnqueue #%d should coalesce (return false) while running", i)
		}
	}

	// wrap runs the task once and, because signals arrived, re-enqueues exactly once.
	sched, err := gocron.NewScheduler(gocron.WithLimitConcurrentJobs(1, gocron.LimitModeWait))
	if err != nil {
		t.Fatal(err)
	}
	sched.Start()
	defer func() { _ = sched.Shutdown() }()

	var ran atomic.Int32
	reruns := make(chan struct{}, 8)
	// Task increments ran; on the re-enqueued run there are no further signals, so it goes idle.
	base := func() error { ran.Add(1); return nil }
	// Manually drive wrap: the first invocation should re-enqueue because rerun is set.
	wrapped := c.wrap(context.Background(), sched, func() error {
		defer func() { reruns <- struct{}{} }()
		return base()
	})
	_ = wrapped() // simulate the queued run completing

	// Exactly one rerun must have been enqueued; wait for it, then confirm no more.
	select {
	case <-reruns:
	case <-time.After(time.Second):
		t.Fatal("expected a re-enqueued run after a coalesced signal")
	}
	select {
	case <-reruns:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("re-enqueued run did not execute")
	}
	// No third run: after the rerun with no new signals, the coalescer goes idle.
	select {
	case <-reruns:
		t.Fatal("unexpected extra run; coalescer should be idle")
	case <-time.After(200 * time.Millisecond):
	}
}

// TestScheduleProcessServices_CoalescesBurst verifies that a burst of enqueue
// signals arriving while a run is in progress collapses to a single follow-up
// run rather than one run per signal.
func TestScheduleProcessServices_CoalescesBurst(t *testing.T) {
	sched, err := gocron.NewScheduler(gocron.WithLimitConcurrentJobs(1, gocron.LimitModeWait))
	if err != nil {
		t.Fatal(err)
	}
	sched.Start()
	defer func() { _ = sched.Shutdown() }()

	gate := make(chan struct{})
	w := &fakeWorker{scheduler: sched, gate: gate, coalescer: &Coalescer{}}
	w.failuresLeft.Store(0) // always succeed

	// First enqueue: starts a run that blocks on the gate.
	if err := ScheduleProcessServices(context.Background(), w); err != nil {
		t.Fatal(err)
	}
	// Wait until the run is actually executing (blocked on gate).
	deadline := time.After(time.Second)
	for w.calls.Load() < 1 {
		select {
		case <-deadline:
			t.Fatal("first run never started")
		case <-time.After(time.Millisecond):
		}
	}

	// Fire many enqueues while the run is blocked: all must coalesce.
	for i := 0; i < 20; i++ {
		if err := ScheduleProcessServices(context.Background(), w); err != nil {
			t.Fatal(err)
		}
	}
	// Release the in-progress run; exactly one coalesced follow-up should run.
	close(gate)

	// Expect exactly 2 total calls: the original + one coalesced rerun.
	time.Sleep(150 * time.Millisecond)
	if got := w.calls.Load(); got != 2 {
		t.Fatalf("ProcessServices called %d times, expected 2 (1 original + 1 coalesced rerun)", got)
	}
}
