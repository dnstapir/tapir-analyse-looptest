package app

import (
	"errors"
    "fmt"
    "sync"

    "github.com/dnstapir/tapir-analyse-looptest/app/ext"
)

type App struct {
	Log           ext.Logger
    Nats          nats
    Tapir         tapir
    Ticker        ticker
	isInitialized bool
    doneChan      chan error
    stopChan      chan bool
    wg            *sync.WaitGroup
}

type nats interface {
    ActivateSubscription() (<-chan string, error) /* Not used */
    Publish(msg string) error
}

type tapir interface {
    GenerateMsg(domain string, flags uint32) (string, error)
}

type ticker interface {
    StartTick() (<-chan int64, error)
    StopTick()
}

func (a *App) Initialize() error {
    var wg sync.WaitGroup
    a.wg = &wg

	a.doneChan = make(chan error, 10)
	a.stopChan = make(chan bool)

	if a.Log == nil {
		return errors.New("no logger object")
	}

	if a.Nats == nil {
		return errors.New("no nats object")
	}


    a.isInitialized = true
    return nil
}

func (a *App) Run() <-chan error {
    if !a.isInitialized {
        panic("app not initialized")
    }


    a.Log.Info("Starting main loop")
    a.wg.Add(1)
    go func() {
        defer a.wg.Done()

        tickCh, err := a.Ticker.StartTick()
        if err != nil {
            a.doneChan <- errors.New("error starting ticker")
	    	return
        }

        for {
	        select {
	        case tick := <-tickCh:
                a.handleTick(tick)
	        case <-a.stopChan:
                a.Log.Info("Stopping main worker thread")
                return
	        }
        }
    }()

    a.Log.Info("Application is now up and running")
	return a.doneChan
}

func (a *App) Stop() error {
    if a.isInitialized {
        a.Log.Info("Stopping application")
    } else {
        a.Log.Info("Stop() called but application was not initialized")
    }

    a.Ticker.StopTick()
    a.stopChan <- true
    a.wg.Wait()

    close(a.doneChan)
    close(a.stopChan)

    a.Log.Info("Application stopped")

	return nil
}

func (a *App) handleTick(tick int64) {
    a.Log.Debug("Received tick '%d'", tick)
    if tick == 0 {
        a.Log.Debug("Tick had zero value, probably garbage. Won't handle...")
        return
    }

    tickDomain := fmt.Sprintf("epoch-%d.ticker.looptest.dnstapir.se.", tick)
    outMsg, err := a.Tapir.GenerateMsg(tickDomain, 2048)
    if err != nil {
        a.Log.Error("Error generating message: %s", err)
        return
    }

    err = a.Nats.Publish(string(outMsg))
    if err != nil {
        a.Log.Error("Error publishing nats message!")
    }
}
