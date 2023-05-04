package cron

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCorn(t *testing.T) {
	fmt.Println("Hello")
	cron := NewCron()
	t1 := &CronTask{
		IsLoop:       true,
		IntervalDura: time.Second * 3,
		Run: func() {
			fmt.Println("T1 Doing...", time.Now())
		},
		Ctx: context.Background(),
	}
	cron.Submit(t1)

	t2 := &CronTask{
		IsLoop:       true,
		IntervalDura: time.Second * 4,
		Run: func() {
			fmt.Println("T2 Doing...", time.Now())
		},
		Ctx: context.Background(),
	}
	cron.Submit(t2)

	t3 := &CronTask{
		IsLoop:       true,
		IntervalDura: time.Second * 10,
		Run: func() {
			fmt.Println("T3 Doing...", time.Now())
		},
		Ctx: context.Background(),
	}
	cron.Submit(t3)

	for {
		time.Sleep(time.Second)
	}
}
