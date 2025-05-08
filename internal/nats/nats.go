package nats

import (
	"github.com/nats-io/nats.go"

    "github.com/dnstapir/tapir-analyse-looptest/app/ext"
)

type Client struct {
    Url        string
    InSubject  string
    OutSubject string
    Queue      string
	Log        ext.Logger
}

func (c Client) ActivateSubscription() (<-chan string, error) {
	nc, err := nats.Connect(c.Url)
	if err != nil {
        c.Log.Error("Error connecting to nats server: %s", err)
		return nil, err
	}

	rawCh := make(chan *nats.Msg)
	_, err = nc.ChanQueueSubscribe(c.InSubject, c.Queue, rawCh)
	if err != nil {
        c.Log.Error("Error subscribing to nats subject: %s", err)
        return nil, err
	}

	strCh := make(chan string)
    go func() {
        for msg := range rawCh {
            c.Log.Debug("Got raw data: %s", string(msg.Data))
            strCh <- string(msg.Data)
            msg.Ack()
        }
        close(strCh)
    }()

	return strCh, nil
}

func (c Client) Publish(msg string) error {
    c.Log.Debug("Attempting to send message: %s")
	nc, err := nats.Connect(c.Url)
	if err != nil {
        c.Log.Error("Error connecting to NATS server")
		return err
	}

	natsMsg := nats.NewMsg(c.OutSubject)
	natsMsg.Data = []byte(msg)

	err = nc.PublishMsg(natsMsg)
	if err != nil {
        c.Log.Error("Error publishing to NATS server")
        return err
	}

    return nil
}
