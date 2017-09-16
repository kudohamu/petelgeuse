package petelgeuse

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type dummyCounter struct {
	mu    *sync.Mutex
	count int
}

func (dc *dummyCounter) countUp() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.count++
}

type dummyTask struct {
	counter *dummyCounter
}

func (d *dummyTask) Run() error {
	d.counter.countUp()
	return nil
}

func TestTask(t *testing.T) {
	testcases := []struct {
		workerSize uint
		taskSize   uint
		expected   int
	}{
		{2, 10, 10},
		{5, 100, 100},
	}

	for _, tc := range testcases {
		counter := &dummyCounter{
			mu: new(sync.Mutex),
		}
		pt := New(&Option{
			WorkerSize: tc.workerSize,
			QueueSize:  tc.workerSize,
		})
		pt.Start()
		for i := uint(0); i < tc.taskSize; i++ {
			pt.Add(&dummyTask{counter: counter})
		}
		pt.Stop()
		assert.Equal(t, tc.expected, counter.count)
	}
}

type dummy3FailTask struct {
	tryCount int8
	counter  *dummyCounter
}

func (d *dummy3FailTask) Run() error {
	d.tryCount++
	if d.tryCount < 4 {
		return errors.New("error")
	}

	d.counter.countUp()
	return nil
}

func TestRetryTask(t *testing.T) {
	testcases := []struct {
		workerSize    uint
		taskSize      uint
		maxRetryCount int16
		expected      int
	}{
		{10, 10, 3, 10},
		{10, 10, 2, 0},
	}

	for _, tc := range testcases {
		counter := &dummyCounter{
			mu: new(sync.Mutex),
		}
		pt := New(&Option{
			WorkerSize:    tc.workerSize,
			QueueSize:     tc.workerSize,
			MaxRetryCount: tc.maxRetryCount,
		})
		pt.Start()
		for i := uint(0); i < tc.taskSize; i++ {
			pt.Add(&dummy3FailTask{
				tryCount: 0,
				counter:  counter,
			})
		}
		pt.Stop()
		assert.Equal(t, tc.expected, counter.count)
	}
}
