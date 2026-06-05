package core

import "time"

// Clock provides deterministic time to evaluators that need now().
type Clock interface {
Now() time.Time
}

type systemClock struct{}

// SystemClock returns a clock backed by time.Now.
func SystemClock() Clock {
return systemClock{}
}

func (systemClock) Now() time.Time {
return time.Now()
}

type fixedClock struct {
time time.Time
}

// FixedClock returns a clock that always reports t.
func FixedClock(t time.Time) Clock {
return fixedClock{time: t}
}

func (c fixedClock) Now() time.Time {
return c.time
}
