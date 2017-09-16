// Package petelgeuse is simple golang worker library.
package petelgeuse

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// Manager represents manager for control workers.
type Manager struct {
	taskQueue chan<- *runner
	ctx       context.Context
	cancel    context.CancelFunc
	workers   []*worker
	wg        *sync.WaitGroup
	mu        *sync.Mutex
	canAdd    bool
	option    *Option
}

// Task represents all sorts of task to let the worker do.
type Task interface {
	Run() error
}

type worker struct {
	taskQueue <-chan *runner
	wg        *sync.WaitGroup
	m         *Manager
}

type runner struct {
	task        Task
	retryCount  int16
	nextBackOff uint // milli second
}

// Option represents optional parameters.
type Option struct {
	WorkerSize int // required
	QueueSize  int // required

	// optional.
	// default: 0 (no retry).
	MaxRetryCount int16

	// optional.
	// default: none, 1000 <= n
	MaxRetryMillisecond uint

	// optional.
	// default: 1, 1000 <= n
	MinRetryMillisecond uint

	// default configuration for backoff. optional.
	// e.g. https://github.com/grpc/grpc/blob/c9d38573721a0ba74eb2fd238876e0a58b1a3c6b/doc/connection-backoff.md
	BackOffMultiplier float64 // default: 1.6, 1 < n
	BackoffJitter     float64 // default: 0.2, 0 < n < 1
}

// New creates maneger instance.
func New(option *Option) *Manager {
	var wg sync.WaitGroup
	var mu sync.Mutex

	// set default values to option
	if option.MinRetryMillisecond < 1000 {
		option.MinRetryMillisecond = 1000
	}
	if option.BackOffMultiplier <= 1 {
		option.BackOffMultiplier = 1.6
	}
	if option.BackoffJitter <= 0 || 1 <= option.BackoffJitter {
		option.BackoffJitter = 0.2
	}

	// create instances.
	tq := make(chan *runner, option.QueueSize)
	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		taskQueue: tq,
		ctx:       ctx,
		cancel:    cancel,
		workers:   make([]*worker, option.WorkerSize),
		wg:        &wg,
		mu:        &mu,
		canAdd:    true,
		option:    option,
	}

	for i := 0; i < option.WorkerSize; i++ {
		m.workers[i] = &worker{
			taskQueue: tq,
			wg:        &wg,
			m:         m,
		}
	}

	return m
}

// Add enqueues a new task.
func (m *Manager) Add(task Task) {
	if !m.canAdd {
		return
	}

	m.wg.Add(1)
	m.taskQueue <- &runner{
		task:        task,
		retryCount:  0,
		nextBackOff: m.option.MinRetryMillisecond,
	}
}

// Start starts each worker.
func (m *Manager) Start() {
	for _, w := range m.workers {
		w.start()
	}
}

// Stop stops all workers.
// Stop pending `return` until all the currently queued tasks are processed.
func (m *Manager) Stop() {
	m.mu.Lock()
	m.canAdd = false
	m.mu.Unlock()

	m.wg.Wait()

	m.cancel()
	close(m.taskQueue)
}

// StopImmediately stops all workers.
// StopImmediately returns as soon as all the tasks currently being processed by the worker, and discard queued tasks.
func (m *Manager) StopImmediately() {
	m.mu.Lock()
	m.canAdd = false
	m.mu.Unlock()

	m.cancel()
	close(m.taskQueue)
}

func (m *Manager) retry(ctx context.Context, r *runner) {
	// backOff
	backOff := r.nextBackOff
	rand.Seed(time.Now().Unix())
	r.nextBackOff = uint(float64(backOff) * m.option.BackOffMultiplier)
	if 1000 < m.option.MaxRetryMillisecond && m.option.MaxRetryMillisecond < r.nextBackOff {
		r.nextBackOff = m.option.MaxRetryMillisecond
	}
	uniformedBackOff := uint(float64(backOff) + float64(backOff)*(rand.Float64()*m.option.BackoffJitter*2-m.option.BackoffJitter))

	select {
	case <-time.After(time.Duration(uniformedBackOff) * time.Millisecond):
		// retry
		m.taskQueue <- r
	case <-ctx.Done():
	}
}

func (w *worker) start() {
	go func() {
		for {
			select {
			case r, ok := <-w.taskQueue:
				if !ok {
					return
				}

				if err := r.task.Run(); err != nil && r.retryCount < w.m.option.MaxRetryCount {
					r.retryCount++
					go w.m.retry(w.m.ctx, r)
					continue
				}
				w.wg.Done()
			case <-w.m.ctx.Done():
				return
			}
		}
	}()
}
