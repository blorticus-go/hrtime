# hrtime
High resolution timer functions in Go

## Rationale

The resolution of the `time` functions in the standard golang library
can have poor resolution (at or greater than 1 ms).  This library aims
to provide higher resolution timer functions.  Currently, it uses the
`hrtimer` syscall interface, which means it will certainly only work on
POSIX systems, and likely not all (for example, it does not work on Darwin/MacOS).

## tick Timer

This module provides a tick timer based on `hrtimer` using a monotonic clock:

```go
package main

import (
	"fmt"
	"time"

	"github.com/blorticus-go/hrtime"
)

func main() {
	ticker := hrtime.NewMonotonicTicker(100 * time.Microsecond)

	if err := ticker.Start(); err != nil {
		panic(err)
	}

	tickCounter := 0
	for {
		<-ticker.C
        go doSomethingUseful()
		tickCounter++
		if tickCounter == 100000 { // 10 seconds have elapsed
			break
		}
	}

	ticker.Stop()
}
```

Keep in mind that a call to the ticker channel is not guaranteed to happen
within a single interval period.  When the resolution is small (as in this
example, with a resolution of 100 µs), the amount of processing time surrounding
the channel functions can sometimes (and at small enough resolutions, often or
always) take longer than resolution period.  The channel returns the number of
ticks that occurred since the last channel read.

It is also important to note that implementations of `hrtimer` don't have infinite
granularity.  On any platform, you will eventually hit the minimum tick limit.