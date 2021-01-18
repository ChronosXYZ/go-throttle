package throttle

import "time"

type Config struct {
	NumTokens   float64
	Delay       time.Duration
	RefillRate  float64
	DefaultCost float64
	Capacity    float64
	MaxCapacity int
}