package hrtime_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/blorticus-go/hrtime"
)

func TestMonotonicTicker(t *testing.T) {
	ticker := hrtime.NewMonotonicTicker(time.Second)

	if err := monotonicTicker10SecondTest(ticker); err != nil {
		t.Fatalf("on first round using the MonotonicTicker: %s", err.Error())
	}
	if err := monotonicTicker10SecondTest(ticker); err != nil {
		t.Errorf("on second round using the MonotonicTicker: %s", err.Error())
	}

	ticker = hrtime.NewMonotonicTicker(500 * time.Millisecond)

	if err := ticker.Start(); err != nil {
		t.Fatalf("on Start() for delay test: %s", err.Error())
	}

	time.Sleep(2 * time.Second)

	ticksSinceLastChannelRead := <-ticker.C
	if ticksSinceLastChannelRead < 3 || ticksSinceLastChannelRead > 5 {
		t.Errorf("on read of channel after 2 second sleep, expected tick count to be in range 3..5, got %d", ticksSinceLastChannelRead)
	}

	ticksSinceLastChannelRead = <-ticker.C
	if ticksSinceLastChannelRead != 1 {
		t.Errorf("on read of channel after no sleep, expected tick count to be 1, got %d", ticksSinceLastChannelRead)
	}

	if err := ticker.Stop(); err != nil {
		t.Errorf("on Stop(): %s", err.Error())
	}
}

func monotonicTicker10SecondTest(ticker *hrtime.MonotonicTicker) error {
	tickCount := 0
	at10Seconds := time.After(10 * time.Second)

	if err := ticker.Start(); err != nil {
		return fmt.Errorf("error on ticker.Start(): %s", err.Error())
	}

	var selectRoundError error
	func() {
		for {
			select {
			case _, open := <-ticker.C:
				if !open {
					selectRoundError = fmt.Errorf("channel is closed")
					return
				}
				tickCount++
			case <-at10Seconds:
				if err := ticker.Stop(); err != nil {
					selectRoundError = fmt.Errorf("on ticker.Stop(), error = (%s)", err.Error())
				}
				return
			}
		}
	}()

	if selectRoundError != nil {
		return selectRoundError
	}

	if tickCount < 9 || tickCount > 10 {
		return fmt.Errorf("expected 9 <= tickCount <= 10, got %d", tickCount)
	}

	return nil
}
