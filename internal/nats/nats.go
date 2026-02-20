package nats

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/dnstapir/tapir-analyse-looptest/internal/common"
)

// TODO find more suitable location to put the analyst identifier
const analystNatsIdentifier string = "tapir-analyse-looptest"

type Conf struct {
	Debug                    bool   `toml:"debug"`
	Url                      string `toml:"url"`
	EventSubject             string `toml:"event_subject"`
	ObservationSubjectPrefix string `toml:"observation_subject_prefix"`
	LooptestBucket           string `toml:"looptest_bucket"`
	Log                      common.Logger
}

type natsClient struct {
	log                      common.Logger
	url                      string
	eventSubject             string
	observationSubjectPrefix string
	looptestBucket           string
	kvLooptest               jetstream.KeyValue
	conn                     *nats.Conn
}

func Create(conf Conf) (*natsClient, error) {
	nc := new(natsClient)

	if conf.Log == nil {
		return nil, errors.New("nil logger")
	}
	nc.log = conf.Log

	if conf.Url == "" {
		return nil, errors.New("no NATS URL")
	}

	if conf.EventSubject == "" {
		return nil, errors.New("no event subject")
	}

	if conf.ObservationSubjectPrefix == "" {
		return nil, errors.New("no observation subject prefix")
	}

	if conf.LooptestBucket == "" {
		return nil, errors.New("no looptest bucket")
	}

	nc.url = conf.Url
	nc.eventSubject = strings.Trim(conf.EventSubject, common.NATS_DELIM)
	nc.observationSubjectPrefix = strings.Trim(conf.ObservationSubjectPrefix, common.NATS_DELIM)
	nc.looptestBucket = conf.LooptestBucket

	err := nc.initNats()
	if err != nil {
		nc.log.Error("Error initializing NATS")
		return nil, err
	}

	nc.log.Debug("NATS debug logging enabled")
	return nc, nil
}

func (nc *natsClient) ActivateSubscription(ctx context.Context) (<-chan common.NatsMsg, error) {
	rawChan := make(chan *nats.Msg, 100) // TODO adjustable buffer?
	sub, err := nc.conn.ChanSubscribe(nc.eventSubject, rawChan)
	if err != nil {
		nc.log.Error("Couldn't subscribe to raw nats channel: '%s'", err)
		return nil, err
	}

	outCh := make(chan common.NatsMsg, 100) // TODO adjustable buffer?
	go func() {
		defer close(outCh)
		defer func() { _ = sub.Unsubscribe() }()
		nc.log.Info("Starting NATS listener loop")
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-rawChan:
				if !ok {
					nc.log.Warning("Incoming NATS channel closed")
					return
				}
				nc.log.Debug("Incoming NATS message on '%s'!", msg.Subject)
				msg.Ack()
				natsMsg := common.NatsMsg{
					Headers: make(map[string]string),
					Data:    msg.Data,
					Subject: msg.Subject,
				}
				for h, v := range msg.Header {
					if slices.Contains(common.NATSHEADERS_DNSTAPIR_ALL, h) {
						natsMsg.Headers[h] = v[0] // TODO use entire slice?
					}
				}
				select {
				case outCh <- natsMsg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	nc.log.Info("Subscribed to '%s'", nc.eventSubject)

	return outCh, nil
}

func (nc *natsClient) SetObservationLooptest(ctx context.Context, domain string) error {
	flipped := _flipDomainName(domain)
	subject := strings.Join(
		[]string{
			nc.observationSubjectPrefix,
			common.OBS_LOOPTEST,
			flipped,
		},
		common.NATS_DELIM)

	_, err := nc.kvLooptest.Put(ctx, subject, []byte(analystNatsIdentifier))
	if err != nil {
		nc.log.Error("Couldn't set key '%s': '%s'", subject, err)
		return err
	}
	return nil
}

func (nc *natsClient) Shutdown() error {
	// TODO impl
	return nil
}

func (nc *natsClient) initNats() error {
	conn, err := nats.Connect(nc.url)
	if err != nil {
		nc.log.Error("Error connecting to nats while setting up KV store: %s", err)
		return err
	}
	js, err := jetstream.New(conn)
	if err != nil {
		nc.log.Error("Error creating jetstream handle: %s", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kvLooptest, err := js.KeyValue(ctx, nc.looptestBucket)
	if err != nil {
		nc.log.Error("Error looking up key value store in NATS: %s", err)
		return err
	}

	nc.kvLooptest = kvLooptest
	nc.conn = conn
	nc.log.Debug("Nats key value store created successfully!")

	return nil
}

func _flipDomainName(domain string) string {
	split := strings.Split(strings.Trim(domain, common.NATS_DELIM), common.NATS_DELIM)
	slices.Reverse(split)
	return strings.Join(split, common.NATS_DELIM)
}

//type Client struct {
//	Url        string
//	InSubject  string
//	OutSubject string
//	Queue      string
//	Log        common.Logger
//}
//
//func (c Client) ActivateSubscription() (<-chan string, error) {
//	nc, err := nats.Connect(c.Url)
//	if err != nil {
//		c.Log.Error("Error connecting to nats server: %s", err)
//		return nil, err
//	}
//
//	rawCh := make(chan *nats.Msg)
//	_, err = nc.ChanQueueSubscribe(c.InSubject, c.Queue, rawCh)
//	if err != nil {
//		c.Log.Error("Error subscribing to nats subject: %s", err)
//		return nil, err
//	}
//
//	strCh := make(chan string)
//	go func() {
//		for msg := range rawCh {
//			c.Log.Debug("Got raw data: %s", string(msg.Data))
//			strCh <- string(msg.Data)
//			msg.Ack()
//		}
//		close(strCh)
//	}()
//
//	return strCh, nil
//}
//
//func (c Client) Publish(msg string) error {
//	c.Log.Debug("Attempting to send message: %s", msg)
//	nc, err := nats.Connect(c.Url)
//	if err != nil {
//		c.Log.Error("Error connecting to NATS server")
//		return err
//	}
//
//	natsMsg := nats.NewMsg(c.OutSubject)
//	natsMsg.Data = []byte(msg)
//
//	err = nc.PublishMsg(natsMsg)
//	if err != nil {
//		c.Log.Error("Error publishing to NATS server")
//		return err
//	}
//
//	return nil
//}
