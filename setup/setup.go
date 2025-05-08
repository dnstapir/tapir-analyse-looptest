package setup

import (
    "github.com/dnstapir/tapir-analyse-looptest/app"
    "github.com/dnstapir/tapir-analyse-looptest/internal/logging"
    "github.com/dnstapir/tapir-analyse-looptest/internal/nats"
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

    natsClient, err := nats.CreateClient(conf.Nats.Url, conf.Nats.InSubject, conf.Nats.OutSubject, conf.Nats.Queue)
    if err != nil {
        return nil, err
    }

	a := app.App{
        Log:  log,
        Nats: natsClient,
    }

	return &a, nil
}
