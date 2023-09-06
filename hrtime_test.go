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
		t.Errorf("on first round using the MonotonicTicker: %s", err.Error())
	}
	if err := monotonicTicker10SecondTest(ticker); err != nil {
		t.Errorf("on second round using the MonotonicTicker: %s", err.Error())
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
					fmt.Println("channel is closed")
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
