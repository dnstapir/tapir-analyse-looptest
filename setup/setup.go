package setup

import (
    "github.com/dnstapir/tapir-analyse-looptest/app"
    "github.com/dnstapir/tapir-analyse-looptest/internal/logging"
    "github.com/dnstapir/tapir-analyse-looptest/internal/nats"
    "github.com/dnstapir/tapir-analyse-looptest/internal/tapir"
)

type AppConf struct {
	Debug bool        `json:"debug"`
	Quiet bool        `json:"quiet"`
    Nats  NatsConfig  `json:"nats"`
}

type NatsConfig struct {
	Url     string    `json:"url"`
	InSubject string  `json:"in_subject"`
	OutSubject string `json:"out_subject"`
	Queue   string    `json:"queue"`
}

func BuildApp(conf AppConf) (*app.App, error) {
    log := logging.Create(conf.Debug, conf.Quiet)

    natsClient := nats.Client {
        Url:        conf.Nats.Url,
        InSubject:  conf.Nats.InSubject,
        OutSubject: conf.Nats.OutSubject,
        Queue:      conf.Nats.Queue,
        Log:        log,
    }

    tapirHandle := tapir.Handle{
        Log: log,
    }

	a := app.App{
        Log:  log,
        Nats: natsClient,
        Tapir: tapirHandle,
    }

	return &a, nil
}
