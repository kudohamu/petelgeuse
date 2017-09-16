# petelgeuse

[![GoDoc](https://godoc.org/github.com/kudohamu/petelgeuse?status.svg)](https://godoc.org/github.com/kudohamu/petelgeuse)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

simple golang worker library.  
inspired by https://medium.com/smsjunk/handling-1-million-requests-per-minute-with-golang-f70ac505fcaa.

## Usage

```go
package main

import (
	"fmt"
	"time"

	"github.com/kudohamu/petelgeuse"
)

type HelloTask struct {}

func (w *HelloTask) Run() error {
	fmt.Println("Hello, petelgeuse!")
	time.Sleep(1 * time.Second)
	return nil
}

func main() {
	// create 10 workers (queue size is 20).
	pt := petelgeuse.New(&petelgeuse.Option{
		WorkerSize: 10,
		QueueSize: 20,
	})
	pt.Start()
	defer pt.Stop()

	for i := 0; i < 20; i++ { // register 20 tasks.
		pt.Add(&HelloTask{})
	}
}
```

### with args

```go
package main

import (
	"fmt"
	"time"

	"github.com/kudohamu/petelgeuse"
)

type HelloTask struct {
	number int // define payloads as you want.
}

func (w *HelloTask) Run() error {
	fmt.Printf("Hello, petelgeuse %d!\n", w.number)
	time.Sleep(1 * time.Second)
	return nil
}

func main() {
	pt := petelgeuse.New(&petelgeuse.Option{
		WorkerSize: 10,
		QueueSize: 20,
	})
	pt.Start()
	defer pt.Stop()

	for i := 0; i < 20; i++ {
    func(number int) {
			pt.Add(&HelloTask{
				number: i,
			})
		}(i)
	}
}
```

### retry

```go
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/kudohamu/petelgeuse"
)

type AlwaysFailTask struct{}

func (t *AlwaysFailTask) Run() error {
	fmt.Println(time.Now().String())
	return errors.New("failed")
}

func main() {
	pt := petelgeuse.New(&petelgeuse.Option{
		WorkerSize:    10,
		QueueSize:     10,
		MaxRetryCount: 5, // retry up to 5 times for each task.
	})
	pt.Start()
	defer pt.Stop()

	pt.Add(&AlwaysFailTask{})
}
```

**outputs**

```shell
2017-09-17 05:07:36.747413961 +0900 JST // first running
2017-09-17 05:07:37.90314243 +0900 JST // 1st retry
2017-09-17 05:07:39.469494268 +0900 JST // 2nd retry
2017-09-17 05:07:42.053893216 +0900 JST // 3rd retry
2017-09-17 05:07:45.811261686 +0900 JST // 4th retry
2017-09-17 05:07:53.497230063 +0900 JST // 5th retry
```
