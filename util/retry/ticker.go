package backoff

import (
	"sync"
	"time"
)

// Ticker holds a channel that delivers `ticks' of a clock at times reported by a BackOff.
//
// Ticks will continue to arrive when the previous operation is still running,
// so operations that take a while to fail could run in quick succession.
type Ticker struct {
	C        <-chan time.Time
	c        chan time.Time
	b        BackOffContext
	stop     chan struct{}
	stopOnce sync.Once
}

// Stop turns off a ticker. After Stop, no more ticks will be sent.
func (t *Ticker) Stop() {
	t.stopOnce.Do(func() { close(t.stop) })
}

func (t *Ticker) run() {
	c := t.c
	defer close(c)

	// Ticker is guaranteed to tick at least once.
	afterC := t.send(time.Now())

	for {
		if afterC == nil {
			return
		}

		select {
		case tick := <-afterC:
			afterC = t.send(tick)
		case <-t.stop:
			t.c = nil // Prevent future ticks from being sent to the channel.
			return
		case <-t.b.Context().Done():
			return
		}
	}
}

func (t *Ticker) send(tick time.Time) <-chan time.Time {
	select {
	case t.c <- tick:
	case <-t.stop:
		return nil
	}

	next := t.b.NextBackOff()
	if next == Stop {
		t.Stop()
		return nil
	}

	return time.After(next)
}
