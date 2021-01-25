package throttle

import (
	"context"
	"github.com/reactivex/rxgo/v2"
	"math"
	"sync"
	"time"
)

type Throttle struct {
	lastTimestamp int64
	queueChan chan *req
	config Config
	cfgMut sync.Mutex
	running bool
	ctx context.Context
}

func NewThrottle(ctx context.Context, cfg *Config) *Throttle {
	if cfg == nil {
		cfg = &Config{
			NumTokens:   0,
			Delay:       1 * time.Millisecond,
			RefillRate:  1000000,
			DefaultCost: 1.000,
			Capacity:    1.000,
		}
	}

	t := &Throttle{
		lastTimestamp: -1,
		queueChan: make(chan *req, cfg.MaxCapacity),
		config: *cfg,
		running: false,
		ctx: ctx,
	}
	t.run()
	return t
}

func (t *Throttle) run() {
	go func() {
		for {
			select {
			case <-t.ctx.Done(): {
				return
			}
			case r:=<-t.queueChan: {
				for {
					var hasEnoughTokens bool
					if t.config.Capacity != 0 {
						hasEnoughTokens = t.config.NumTokens > 0
					} else {
						hasEnoughTokens = t.config.NumTokens >= 0
					}
					if hasEnoughTokens {
						var cost float64 = 0
						if r.cost != -1 {
							cost = r.cost
						} else {
							cost = t.config.DefaultCost
						}
						if t.config.NumTokens >= math.Min(cost, t.config.Capacity) {
							t.config.NumTokens -= cost
							r.resolveChan <- rxgo.Of(true)
							close(r.resolveChan)
							break
						}
					}

					if t.lastTimestamp == -1 {
						t.lastTimestamp = time.Now().UnixNano()
					}
					now := time.Now().UnixNano()
					elapsed := now - t.lastTimestamp
					t.lastTimestamp = now
					t.config.NumTokens = math.Min(t.config.Capacity, t.config.NumTokens+float64(elapsed)*t.config.RefillRate)
					time.Sleep(t.config.Delay)
				}
			}
			}
		}
	}()

}

func (t *Throttle) Take(rateLimit time.Duration, cost float64) rxgo.Observable {
	t.cfgMut.Lock()
	defer t.cfgMut.Unlock()
	ch := make(chan rxgo.Item)
	t.queueChan <- &req{
		cost: cost,
		resolveChan: ch,
	}
	if rateLimit == 0 {
		t.config.RefillRate = 0
	} else {
		t.config.RefillRate = float64(1 / rateLimit.Nanoseconds())
	}
	return rxgo.FromChannel(ch)
}

type req struct {
	cost float64
	resolveChan chan rxgo.Item
}

