package queue

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Task struct {
	Handler func(v ...interface{})
	Params  []interface{}
}
type Queue struct {
	capacity       uint64
	runningWorkers uint64
	status         int64
	chTask         chan *Task
	PanicHandler   func(interface{})
	sync.Mutex
}

func NewPool(capacity uint64) (*Queue, error) {
	if capacity <= 0 {
		return nil, errors.New("invalid pool cap")
	}
	q := &Queue{
		capacity: capacity,
		status:   1,
		chTask:   make(chan *Task, capacity),
		PanicHandler: func(i interface{}) {
			fmt.Println("panic recovered:", i)
		},
	}
	return q, nil
}
func (q *Queue) checkWorker() {
	q.Lock()
	defer q.Unlock()
	if q.runningWorkers == 0 && len(q.chTask) > 0 {
		q.run()
	}
}
func (q *Queue) GetCap() uint64 {
	return q.capacity
}
func (q *Queue) GetRunningWorkers() uint64 {
	return atomic.LoadUint64(&q.runningWorkers)
}
func (q *Queue) incRunning() {
	atomic.AddUint64(&q.runningWorkers, 1)
}
func (q *Queue) decRunning() {
	atomic.AddUint64(&q.runningWorkers, ^uint64(0))
}
func (q *Queue) Put(task *Task) error {
	q.Lock()
	defer q.Unlock()
	if q.status == 0 {
		return errors.New("pool already closed")
	}
	if q.GetRunningWorkers() < q.GetCap() {
		q.run()
	}
	if q.status == 1 {
		q.chTask <- task
	}
	return nil
}
func (q *Queue) run() {
	q.incRunning()
	go func() {
		defer func() {
			q.decRunning()
			if r := recover(); r != nil {
				if q.PanicHandler != nil {
					q.PanicHandler(r)
				} else {
					fmt.Printf("Worker panic: %s\n", r)
				}
			}
			q.checkWorker()
		}()
		for {
			select {
			case task, ok := <-q.chTask:
				if !ok {
					return
				}
				task.Handler(task.Params...)
			}
		}
	}()
}
func (q *Queue) setStatus(status int64) bool {
	q.Lock()
	defer q.Unlock()
	if q.status == status {
		return false
	}
	q.status = status
	return true
}
func (q *Queue) close() {
	q.Lock()
	defer q.Unlock()
	close(q.chTask)
}
func (q *Queue) Close() {
	if !q.setStatus(0) {
		return
	}
	for len(q.chTask) > 0 {
		time.Sleep(1e6)
	}
	q.close()
}
