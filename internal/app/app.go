package app

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/dnstapir/tapir-analyse-looptest/internal/common"
)

const c_N_HANDLERS = 3
const c_NATS_DELIM = common.NATS_DELIM

type Conf struct {
	Debug               bool   `toml:"debug"`
	Interval            int    `toml:"interval"`
	LooptestMatchSuffix string `toml:"looptest_match_suffix"`
	Log                 common.Logger
	NatsHandle          nats
	LibtapirHandle      libtapir
}

type appHandle struct {
	id                  string
	looptestMatchSuffix string
	log                 common.Logger
	natsHandle          nats
	libtapirHandle      libtapir
	ticker              *time.Ticker
	exitCh              chan<- common.Exit
	pm
}

type pm struct {
}

type job struct {
	isTick   bool
	tickData int64
	msg      common.NatsMsg
}

type nats interface {
	ActivateSubscription(context.Context) (<-chan common.NatsMsg, error)
	SetObservationLooptest(context.Context, string) error
	Shutdown() error
}

type libtapir interface {
	ExtractDomain([]byte) (string, error)
}

func Create(conf Conf) (*appHandle, error) {
	a := new(appHandle)
	a.id = "main app"

	if conf.Log == nil {
		return nil, common.ErrBadHandle
	}
	a.log = conf.Log

	if conf.NatsHandle == nil {
		return nil, common.ErrBadHandle
	}
	a.natsHandle = conf.NatsHandle

	if conf.LibtapirHandle == nil {
		return nil, common.ErrBadHandle
	}
	a.libtapirHandle = conf.LibtapirHandle

	if conf.Interval > 0 {
		a.ticker = time.NewTicker(time.Duration(conf.Interval) * time.Second)
	} else {
		a.ticker = time.NewTicker(time.Duration(math.MaxInt32) * time.Second)
		a.ticker.Stop()
		a.log.Warning("No interval set. Won't run ticker looptests.")
	}

	if conf.LooptestMatchSuffix == "" {
		a.log.Warning("No match suffix given for looptests. Will only run ticker tests")
	}
	a.looptestMatchSuffix = strings.ToLower(strings.Trim(conf.LooptestMatchSuffix, "."))

	a.log.Debug("Main app debug logging enabled")
	return a, nil
}

func (a *appHandle) Run(ctx context.Context, exitCh chan<- common.Exit) {
	defer a.ticker.Stop()

	var natsChan <-chan common.NatsMsg
	var err error
	a.id = "main app"
	a.exitCh = exitCh
	jobChan := make(chan job, 10)

	if a.looptestMatchSuffix != "" {
		natsChan, err = a.natsHandle.ActivateSubscription(ctx)
		if err != nil {
			a.log.Error("Couldn't activate NATS subscription: '%s'", err)
			a.exitCh <- common.Exit{ID: a.id, Err: err}
			return
		}
	}

	var wg sync.WaitGroup
	for range c_N_HANDLERS {
		wg.Go(func() {
			for j := range jobChan {
				a.handleJob(ctx, j)
			}
			a.log.Info("Worker done!")
		})
	}

MAIN_APP_LOOP:
	for {
		select {
		case t := <-a.ticker.C:
			a.log.Debug("Tick")
			j := job{
				isTick:   true,
				tickData: t.Unix(),
			}
			jobChan <- j
		case natsMsg, ok := <-natsChan:
			if !ok {
				a.log.Warning("NATS channel closed")
				natsChan = nil
			} else {
				a.log.Debug("Incoming NATS message")
				j := job{
					msg: natsMsg,
				}
				jobChan <- j
			}
		case <-ctx.Done():
			a.log.Info("Stopping main worker thread")
			break MAIN_APP_LOOP
		}
	}

	close(jobChan)

	wg.Wait()

	err = a.natsHandle.Shutdown()
	if err != nil {
		a.log.Error("Encountered '%s' during NATS shutdown", err)
	}

	a.exitCh <- common.Exit{ID: a.id, Err: err}
	a.log.Info("Main app shutdown done")
	return
}

func (a *appHandle) handleJob(ctx context.Context, j job) {
	if j.isTick {
		a.handleTick(ctx, j.tickData)
	} else {
		a.handleMsg(ctx, j.msg)
	}
}

func (a *appHandle) handleTick(ctx context.Context, epoch int64) {
	a.log.Debug("Received tick '%d'", epoch)
	if epoch == 0 {
		a.log.Debug("Tick had zero value, probably garbage. Won't handle...")
		return
	}

	tickDomain := fmt.Sprintf("epoch-%d.ticker.looptest.dnstapir.se.", epoch)
	err := a.natsHandle.SetObservationLooptest(ctx, tickDomain)
	if err != nil {
		a.log.Error("Error setting looptest observation for '%s': %s", tickDomain, err)
	}
}

func (a *appHandle) handleMsg(ctx context.Context, msg common.NatsMsg) {
	a.log.Debug("Handling %d byte message on subject %s", len(msg.Data), msg.Subject)
	if len(msg.Data) <= 0 {
		a.log.Warning("Msg had no data, probably garbage. Won't handle...")
		return
	}

	// TODO filter based on NATS headers so not everyone can do a looptest?
	// TODO schema validation

	if a.looptestMatchSuffix == "" {
		a.log.Warning("No looptest match suffix provided, should not be receiving NATS messages.")
		return
	}

	msgDomain, err := a.libtapirHandle.ExtractDomain(msg.Data)
	if err != nil {
		a.log.Error("Error reading domain from message: %s", err)
		return
	}

	if !strings.HasSuffix(msgDomain, "."+a.looptestMatchSuffix) {
		return
	}

	err = a.natsHandle.SetObservationLooptest(ctx, msgDomain)
	if err != nil {
		a.log.Error("Error setting looptest observation for '%s': %s", msgDomain, err)
	} else {
		a.log.Debug("Looptest observation set for '%s'", msgDomain)
	}
}

//type App struct {
//	Log           common.Logger
//	Nats          nats
//	Tapir         tapir
//	Ticker        ticker
//	isInitialized bool
//	doneChan      chan error
//	stopChan      chan bool
//	wg            *sync.WaitGroup
//}
//
//type nats interface {
//	ActivateSubscription() (<-chan string, error)
//	Publish(msg string) error
//}
//
//type tapir interface {
//	GenerateMsg(domain string, flags uint32) (string, error)
//	ExtractDomain(msgJson string) (string, error)
//}
//
//type ticker interface {
//	StartTick() (<-chan int64, error)
//	StopTick()
//}
//
//func (a *App) Initialize() error {
//	var wg sync.WaitGroup
//	a.wg = &wg
//
//	a.doneChan = make(chan error, 10)
//	a.stopChan = make(chan bool)
//
//	if a.Log == nil {
//		return errors.New("no logger object")
//	}
//
//	if a.Nats == nil {
//		return errors.New("no nats object")
//	}
//
//	if a.Tapir == nil {
//		return errors.New("no tapir object")
//	}
//
//	a.isInitialized = true
//	return nil
//}
//
//func (a *App) Run() <-chan error {
//	if !a.isInitialized {
//		panic("app not initialized")
//	}
//
//	a.Log.Info("Starting main loop")
//	a.wg.Add(1)
//	go func() {
//		defer a.wg.Done()
//
//		tickCh, err := a.Ticker.StartTick()
//		if err != nil {
//			a.doneChan <- errors.New("error starting ticker")
//			return
//		}
//
//        msgChan, err := a.Nats.ActivateSubscription()
//        if err != nil {
//            a.doneChan <- errors.New("error activating nats subscription")
//            return
//        }
//
//		for {
//			select {
//			case tick := <-tickCh:
//				a.handleTick(tick)
//			case msg := <-msgChan:
//				a.handleMsg(msg)
//			case <-a.stopChan:
//				a.Log.Info("Stopping main worker thread")
//				return
//			}
//		}
//	}()
//
//	a.Log.Info("Application is now up and running")
//	return a.doneChan
//}
//
//func (a *App) Stop() error {
//	if a.isInitialized {
//		a.Log.Info("Stopping application")
//	} else {
//		a.Log.Info("Stop() called but application was not initialized")
//	}
//
//	a.Ticker.StopTick()
//	a.stopChan <- true
//	a.wg.Wait()
//
//	close(a.doneChan)
//	close(a.stopChan)
//
//	a.Log.Info("Application stopped")
//
//	return nil
//}
//
//func (a *App) handleTick(tick int64) {
//	a.Log.Debug("Received tick '%d'", tick)
//	if tick == 0 {
//		a.Log.Debug("Tick had zero value, probably garbage. Won't handle...")
//		return
//	}
//
//	tickDomain := fmt.Sprintf("epoch-%d.ticker.looptest.dnstapir.se.", tick)
//	outMsg, err := a.Tapir.GenerateMsg(tickDomain, 2048)
//	if err != nil {
//		a.Log.Error("Error generating message: %s", err)
//		return
//	}
//
//	err = a.Nats.Publish(string(outMsg))
//	if err != nil {
//		a.Log.Error("Error publishing nats message!")
//	}
//}
//
//func (a *App) handleMsg(msg string) {
//	a.Log.Debug("Received message '%s'", msg)
//	if msg == "" {
//		a.Log.Debug("Msg had zero value, probably garbage. Won't handle...")
//		return
//	}
//
//	msgDomain, err := a.Tapir.ExtractDomain(msg)
//	if err != nil {
//		a.Log.Error("Error reading domain from message: %s", err)
//		return
//	}
//
//    if !strings.HasSuffix(msgDomain, "from-edge.looptest.dnstapir.se.") {
//		a.Log.Debug("Ignoring msg with domain '%s'", msgDomain)
//        return
//    }
//
//	outMsg, err := a.Tapir.GenerateMsg(msgDomain, 2048)
//	if err != nil {
//		a.Log.Error("Error generating message: %s", err)
//		return
//	}
//
//	err = a.Nats.Publish(string(outMsg))
//	if err != nil {
//		a.Log.Error("Error publishing nats message!")
//	}
//}
