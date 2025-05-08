package nats

import (
	"github.com/nats-io/nats.go"
)

type Client struct {
    url string
    inSubject string
    outSubject string
    queue string
}

func CreateClient(url, inSubject, outSubject, queue string) (Client, error) {
    return Client{url, inSubject, outSubject, queue}, nil
}

func (c Client) ActivateSubscription() (<-chan string, error) {
	nc, err := nats.Connect(c.url)
	if err != nil {
		return nil, err
	}

	rawCh := make(chan *nats.Msg)
	_, err = nc.ChanQueueSubscribe(c.inSubject, c.queue, rawCh)
	if err != nil {
        return nil, err
	}

	strCh := make(chan string)
    go func() {
        for msg := range rawCh {
            strCh <- string(msg.Data)
            msg.Ack()
        }
        close(strCh)
    }()

	return strCh, nil
}

func (c Client) Publish(msg string) error {
	nc, err := nats.Connect(c.url)
	if err != nil {
		return err
	}

	natsMsg := nats.NewMsg(c.outSubject)
	natsMsg.Data = []byte(msg)

	err = nc.PublishMsg(natsMsg)
	if err != nil {
        return err
	}

    return nil
}
