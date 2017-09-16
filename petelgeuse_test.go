package petelgeuse

import (
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
		workerSize int
		taskSize   int
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
		for i := 0; i < tc.taskSize; i++ {
			pt.Add(&dummyTask{counter: counter})
		}
		pt.Stop()
		assert.Equal(t, tc.expected, counter.count)
	}
}
