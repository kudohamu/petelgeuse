// Package petelgeuse is simple golang worker library.
package petelgeuse

import "sync"

// Manager represents manager for control workers.
type Manager struct {
	taskQueue chan<- Task
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
	taskQueue <-chan Task
	wg        *sync.WaitGroup
}

// Option represents optional parameters.
type Option struct {
	WorkerSize int // required
	QueueSize  int // required
}

// New creates maneger instance.
func New(option *Option) *Manager {
	var wg sync.WaitGroup
	var mu sync.Mutex
	tq := make(chan Task, option.QueueSize)

	m := &Manager{
		taskQueue: tq,
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
		}
	}

	return m
}

// Add enqueue a new task.
func (m *Manager) Add(task Task) {
	if !m.canAdd {
		return
	}

	m.wg.Add(1)
	m.taskQueue <- task
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
	close(m.taskQueue)
}

// StopImmediately stops all workers.
// StopImmediately returns as soon as all the tasks currently being processed by the worker, and discard queued tasks.
func (m *Manager) StopImmediately() {
	m.mu.Lock()
	m.canAdd = false
	m.mu.Unlock()

	close(m.taskQueue)
}

func (w *worker) start() {
	go func() {
		for {
			select {
			case task, ok := <-w.taskQueue:
				if !ok {
					return
				}
				task.Run()
				w.wg.Done()
			}
		}
	}()
}
