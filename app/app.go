package app

import (
	"errors"

    "github.com/dnstapir/tapir-analyse-looptest/app/ext"
)

type App struct {
	Log       ext.Logger
    Nats      nats
    Tapir     tapir
	isRunning bool
    doneChan  chan error
    stopChan  chan bool
}

type nats interface {
    ActivateSubscription() (<-chan string, error)
    Publish(msg string) error
}

type tapir interface {
    GenerateMsg(domain string, flags uint32) (string, error)
}

func (a *App) Run() <-chan error {
	a.doneChan = make(chan error, 10)
	a.stopChan = make(chan bool, 1)
	a.isRunning = true

	if a.Log == nil {
		a.doneChan <- errors.New("no logger object")
		a.isRunning = false
	}

	if a.Nats == nil {
		a.doneChan <- errors.New("no nats object")
		a.isRunning = false
	}

    msgChan, err := a.Nats.ActivateSubscription()
    if err != nil {
        a.doneChan <- errors.New("error activating nats subscription")
        a.isRunning = false
    }

    go func() {
        for {
	        select {
	        case msg := <-msgChan:
                a.handleMsg(msg)
	        case <-a.stopChan:
                a.Log.Info("Stopping main worker thread")
	        }
        }
    }()

	return a.doneChan
}

func (a *App) Stop() error {
    if a.Log != nil {
        if a.isRunning {
            a.Log.Info("Stopping application")
        } else {
            a.Log.Info("Stop() called but application was not running")
        }
    }

    a.stopChan <- true

    close(a.doneChan)
    close(a.stopChan)

	return nil
}

func (a *App) handleMsg(msg string) {
    a.Log.Debug("Received message '%s'", msg)

    outMsg, err := a.Tapir.GenerateMsg(msg, 2048)
    if err != nil {
        a.Log.Error("Error generating message: %s", err)
        return
    }

    err = a.Nats.Publish(string(outMsg))
    if err != nil {
        a.Log.Error("Error publishing nats message!")
    }
}
