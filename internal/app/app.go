package app

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/dnstapir/tapir-analyse-lib/common"
	"github.com/dnstapir/tapir-analyse-lib/libtapir"
)

const c_N_HANDLERS = 3
const c_NATS_DELIM = common.NATS_DELIM

type Conf struct {
	Debug               bool   `toml:"debug"`
	Interval            int    `toml:"interval"`
	LooptestMatchSuffix string `toml:"looptest_match_suffix"`
	AnalystID           string
	Log                 common.Logger
	NatsHandle          nats
}

type appHandle struct {
	id                  string
	looptestMatchSuffix string
	log                 common.Logger
	natsHandle          nats
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
	SetObservation(context.Context, string, string) error
	Shutdown() error
}

func Create(conf Conf) (*appHandle, error) {
	a := new(appHandle)

	if conf.Log == nil {
		return nil, common.ErrBadHandle
	}
	a.log = conf.Log

	if conf.NatsHandle == nil {
		return nil, common.ErrBadHandle
	}
	a.natsHandle = conf.NatsHandle

	if conf.AnalystID == "" {
		a.log.Error("Bad analyst ID when creating looptest")
		return nil, common.ErrBadParam
	}
	a.id = conf.AnalystID

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
	a.looptestMatchSuffix = libtapir.NormalizeDomainNameSuffix(conf.LooptestMatchSuffix)

	a.log.Debug("Main app debug logging enabled")
	return a, nil
}

func (a *appHandle) Run(ctx context.Context, exitCh chan<- common.Exit) {
	defer a.ticker.Stop()

	var natsChan <-chan common.NatsMsg
	var err error
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
	err := a.natsHandle.SetObservation(ctx, tickDomain, common.OBS_LOOPTEST)
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

	msgDomain, err := libtapir.ExtractDomain(msg.Data)
	if err != nil {
		a.log.Error("Error reading domain from message: %s", err)
		return
	}

	if !strings.HasSuffix(msgDomain, a.looptestMatchSuffix) {
		a.log.Debug("No match: %s", msgDomain)
		return
	} else {
		a.log.Debug("Looptest domain match: %s", msgDomain)
	}

	err = a.natsHandle.SetObservation(ctx, msgDomain, common.OBS_LOOPTEST)
	if err != nil {
		a.log.Error("Error setting looptest observation for '%s': %s", msgDomain, err)
	} else {
		a.log.Debug("Looptest observation set for '%s'", msgDomain)
	}
}
