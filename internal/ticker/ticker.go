package ticker

import (
	"sync"
	"time"

	"github.com/dnstapir/tapir-analyse-looptest/app/ext"
)

type Handle struct {
	Log      ext.Logger
	ticker   *time.Ticker
	stopChan chan bool
	wg       *sync.WaitGroup
}

func (h *Handle) StartTick() (<-chan int64, error) {
	epochTimestampCh := make(chan int64)
	var wg sync.WaitGroup
	h.stopChan = make(chan bool)
	h.wg = &wg
	h.ticker = time.NewTicker(time.Minute)

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		defer close(epochTimestampCh)
		defer h.ticker.Stop()
		h.Log.Debug("Starting ticker loop")
		for {
			select {
			case timestampRaw := <-h.ticker.C:
				h.Log.Debug("Tick: %s", timestampRaw)
				epochTimestampCh <- timestampRaw.Unix()
			case <-h.stopChan:
				h.Log.Debug("Ticker loop returning")
				return
			}
		}
	}()

	return epochTimestampCh, nil
}

func (h *Handle) StopTick() {
	h.Log.Info("Stopping ticker")
	h.stopChan <- true
	h.wg.Wait()
	close(h.stopChan)
	h.Log.Info("Ticker stopped")
}
