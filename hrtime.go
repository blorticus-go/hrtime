package hrtime

// hrtime aims to provide timer functions with higher resolution than the standard golang time library.

import (
	"fmt"
	"os"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// The ticker read loop executes in a goroutine and detects that Stop()
// has happened on a read error (because the connected file descriptor is
// closed in Stop()).  However, a call to Start() after Stop() may create
// a new file handle and a new channel before the read error is detected
// in the previous goroutine, leading to a race condition when trying to
// close the channel and filehandle.  So, when they are created, they are
// tied to a tickerHandles struct with a synchronized closer.  When Start()
// creates a new channel and a new filehandle, a new tickerHandles object is
// also created.  Meanwhile, the previously running goroutine will hold a reference
// to the previous set of handles.
type tickerHandles struct {
	tickFile      *os.File
	sharedChannel chan uint64
	mu            sync.Mutex
	areClosed     bool
}

func (c *tickerHandles) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.areClosed {
		close(c.sharedChannel)
		c.tickFile.Close()
		c.areClosed = true
	}
}

// A MonotonicTicker is a ticker using a monotonic clock.
type MonotonicTicker struct {
	// After the ticker is started (using Start()), periodic
	// writes will occur on this channel.  The value is the
	// approximate number of ticks that have occurred since the
	// last channel read.  However, while the channel blocks
	// for the receiver, it does not block from the ticker, so
	// if a read is missed, one or more ticks may be lost.
	C               chan uint64
	desiredInterval time.Duration
	mu              sync.Mutex
	handles         *tickerHandles
	inStoppedState  bool
}

// NewMonotonicTicker creates a ticker that is intended to fire a tick
// near every interval.
func NewMonotonicTicker(interval time.Duration) *MonotonicTicker {
	return &MonotonicTicker{
		desiredInterval: interval,
		inStoppedState:  true,
	}
}

// Start starts the ticker.  Writes to ticker.C should now occur according
// to the interval configured in the constructor.  Each time Start() is run,
// the ticker.C channel is replaced with a new one.  Once a ticker is started,
// Start() cannot be run again until Stop() is run on the ticker.
func (ticker *MonotonicTicker) Start() error {
	ticker.mu.Lock()
	defer ticker.mu.Unlock()

	if !ticker.inStoppedState {
		return fmt.Errorf("must Stop() before performing Start() again")
	}

	fd, err := unix.TimerfdCreate(unix.CLOCK_MONOTONIC, unix.TFD_NONBLOCK|unix.TFD_CLOEXEC)
	if err != nil {
		return err
	}

	timerFile := os.NewFile(uintptr(fd), "timerfd")

	itimerSpec := &unix.ItimerSpec{
		Value:    unix.NsecToTimespec(ticker.desiredInterval.Nanoseconds()),
		Interval: unix.NsecToTimespec(ticker.desiredInterval.Nanoseconds()),
	}

	if err := settimeUsingFile(timerFile, itimerSpec); err != nil {
		timerFile.Close()
		return err
	}

	ticker.C = make(chan uint64)

	ticker.handles = &tickerHandles{
		tickFile:      timerFile,
		sharedChannel: ticker.C,
	}

	go monotonicTickerReadLoop(ticker.handles)

	return nil
}

func monotonicTickerReadLoop(handles *tickerHandles) {
	b := make([]byte, 8)

	ticksSinceLastChannelRead := uint64(0)
	for {
		bytesRead, err := handles.tickFile.Read(b)
		if bytesRead != 8 || err != nil {
			handles.close()
			return
		}

		// read bytes are in host byte order
		ticksSinceLastChannelRead += *(*uint64)(unsafe.Pointer(&b[0]))

		select {
		case handles.sharedChannel <- ticksSinceLastChannelRead:
			ticksSinceLastChannelRead = 0
		default:
		}
	}
}

// Stop stops a running ticker.  The associated channel will be closed
// from the ticker side.
func (ticker *MonotonicTicker) Stop() error {
	ticker.mu.Lock()
	handles := ticker.handles
	ticker.inStoppedState = true
	ticker.mu.Unlock()

	handles.close()

	return nil
}

func settimeUsingFile(f *os.File, itimerSpec *unix.ItimerSpec) error {
	raw, err := f.SyscallConn()
	if err != nil {
		return err
	}

	var fdSettimeError error
	err = raw.Control(func(fdInControl uintptr) {
		fdSettimeError = unix.TimerfdSettime(int(fdInControl), 0, itimerSpec, nil)
	})

	if fdSettimeError != nil {
		return fdSettimeError
	}
	if err != nil {
		return err
	}

	return nil
}
