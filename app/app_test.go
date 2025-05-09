package app

import (
	"testing"

    "github.com/dnstapir/tapir-analyse-looptest/app/ext"
)

func setupApp() *App {
	a := App{
        Log:  ext.FakeLogger{},
        Ticker: fakeTicker{},
        Tapir: fakeTapir{},
        Nats: mockNats{},
    }

    return &a
}

func TestAppBasic(t *testing.T) {
    application := setupApp()


    err := application.Initialize()
    if err != nil {
        t.Fatalf("Error initializing app: %s", err)
    }

    //err = application.Stop()
    //if err != nil {
    //    t.Fatalf("Error stopping app: %s", err)
    //}
}

type fakeTicker struct{}

func (ft fakeTicker) StartTick() (<-chan int64, error) {
    return nil, nil
}

func (ft fakeTicker) StopTick() {
}

type fakeTapir struct{}

func (ft fakeTapir) GenerateMsg(domain string, flags uint32) (string, error) {
    return "fake-tapir", nil
}

type mockNats struct{}

func (mn mockNats) ActivateSubscription() (<-chan string, error) {
    return nil, nil
}

func (mn mockNats) Publish(msg string) error {
    return nil
}
