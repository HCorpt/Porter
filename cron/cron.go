package cron

import (
	"container/heap"
	"context"
	"time"
)

type CronTask struct {
	IsLoop       bool
	ExecTime     time.Time
	IntervalDura time.Duration
	Run          func()
	Ctx          context.Context
}

type CronTaskPriorityQueue []*CronTask

func (pq CronTaskPriorityQueue) Len() int {
	return len(pq)
}

func (pq CronTaskPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *CronTaskPriorityQueue) Push(c any) {
	*pq = append(*pq, c.(*CronTask))
}

func (pq *CronTaskPriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[0 : n-1]
	return item
}

func (pq CronTaskPriorityQueue) Less(i, j int) bool {
	return pq[i].ExecTime.Before(pq[j].ExecTime)
}

type Cron struct {
	submitChan chan *CronTask
	exit       chan struct{}
	pq         CronTaskPriorityQueue
}

func NewCron() *Cron {
	c := &Cron{
		submitChan: make(chan *CronTask, 100),
		exit:       make(chan struct{}),
		pq:         make(CronTaskPriorityQueue, 0),
	}
	go c.Loop()
	return c
}

func (c *Cron) Submit(t *CronTask) {
	c.submitChan <- t
}

func (c *Cron) Loop() {
	for {
		waitTime := time.Hour * 24 // one day
		now := time.Now()
		// DoTask
		for len(c.pq) > 0 {
			if !c.pq[0].ExecTime.After(now) {
				t := heap.Pop(&c.pq).(*CronTask)
				go c.AsyncDoTask(t)
			} else {
				break
			}
		}
		// ReCalculate NextTime
		if len(c.pq) != 0 {
			waitTime = c.pq[0].ExecTime.Sub(now)
		}
		select {
		case t := <-c.submitChan:
			heap.Push(&c.pq, t)
		case <-time.After(waitTime):
		case <-c.exit:
			return
		}
	}
}

func (c *Cron) AsyncDoTask(t *CronTask) {
	select {
	case <-t.Ctx.Done():
		return
	default:
		t.Run()
		if t.IsLoop && t.IntervalDura != 0 {
			t.ExecTime = time.Now().Add(t.IntervalDura)
			c.Submit(t)
		}
	}
}

func (c *Cron) Exit() {
	close(c.exit)
}
