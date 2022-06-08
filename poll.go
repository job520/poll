package poll

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
type Pool struct {
	capacity       uint64
	runningWorkers uint64
	status         int64
	chTask         chan *Task
	PanicHandler   func(interface{})
	sync.Mutex
}

func NewPool(capacity uint64) (*Pool, error) {
	if capacity <= 0 {
		return nil, errors.New("invalid pool cap")
	}
	p := &Pool{
		capacity: capacity,
		status:   1,
		chTask:   make(chan *Task, capacity),
		PanicHandler: func(i interface{}) {
			fmt.Println("panic recovered:", i)
		},
	}
	return p, nil
}
func (p *Pool) checkWorker() {
	p.Lock()
	defer p.Unlock()
	if p.runningWorkers == 0 && len(p.chTask) > 0 {
		p.run()
	}
}
func (p *Pool) GetCap() uint64 {
	return p.capacity
}
func (p *Pool) GetRunningWorkers() uint64 {
	return atomic.LoadUint64(&p.runningWorkers)
}
func (p *Pool) incRunning() {
	atomic.AddUint64(&p.runningWorkers, 1)
}
func (p *Pool) decRunning() {
	atomic.AddUint64(&p.runningWorkers, ^uint64(0))
}
func (p *Pool) Put(task *Task) error {
	p.Lock()
	defer p.Unlock()
	if p.status == 0 {
		return errors.New("pool already closed")
	}
	if p.GetRunningWorkers() < p.GetCap() {
		p.run()
	}
	if p.status == 1 {
		p.chTask <- task
	}
	return nil
}
func (p *Pool) run() {
	p.incRunning()
	go func() {
		defer func() {
			p.decRunning()
			if r := recover(); r != nil {
				if p.PanicHandler != nil {
					p.PanicHandler(r)
				} else {
					fmt.Printf("Worker panic: %s\n", r)
				}
			}
			p.checkWorker()
		}()
		for {
			select {
			case task, ok := <-p.chTask:
				if !ok {
					return
				}
				task.Handler(task.Params...)
			}
		}
	}()
}
func (p *Pool) setStatus(status int64) bool {
	p.Lock()
	defer p.Unlock()
	if p.status == status {
		return false
	}
	p.status = status
	return true
}
func (p *Pool) close() {
	p.Lock()
	defer p.Unlock()
	close(p.chTask)
}
func (p *Pool) Close() {
	if !p.setStatus(0) {
		return
	}
	for len(p.chTask) > 0 {
		time.Sleep(1e6)
	}
	p.close()
}
