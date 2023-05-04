package utils

import "time"

type Stat struct {
	Time time.Time   `json:"time"`
	Info interface{} `json:"info"`
}

type RingStats struct {
	Stats   []Stat `json:"stats"`
	MaxSize int    `json:"-"`
}

func NewRingStats(n int) *RingStats {
	return &RingStats{
		Stats:   make([]Stat, 0, n),
		MaxSize: n,
	}
}

func (r *RingStats) Append(s Stat) {
	r.Stats = append(r.Stats, s)
	if len(r.Stats) > r.MaxSize {
		r.Stats = r.Stats[len(r.Stats)-r.MaxSize:]
	}
}
