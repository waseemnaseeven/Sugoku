package main 

import (
	"time"
)

type Stopwatch struct {
	running bool
	start   time.Time
	elapsed time.Duration
}

func (sw *Stopwatch) Start() {
	if sw.running {
		return
	}
	sw.running = true
	sw.start = time.Now()
}

func (sw *Stopwatch) Stop() {
	if !sw.running {
		return
	}
	sw.elapsed += time.Since(sw.start)
	sw.running = false
}

func (sw *Stopwatch) Reset() {
	sw.running = false
	sw.elapsed = 0
}

// Real time elapsed while solver is active
func (sw *Stopwatch) Elapsed() time.Duration {
	if sw.running {
		return sw.elapsed + time.Since(sw.start)
	}
	return sw.elapsed
}