package tapir

import (
	"encoding/json"
    "strings"
	"time"

	"github.com/dnstapir/tapir"
    "github.com/dnstapir/edm/pkg/protocols"

	"github.com/dnstapir/tapir-analyse-looptest/app/ext"
)

type Handle struct {
	Log ext.Logger
}

func (h Handle) GenerateMsg(domainStr string, flags uint32) (string, error) {
	domain := tapir.Domain{
		Name:         domainStr,
		TimeAdded:    time.Now(),
		TTL:          3600,
		TagMask:      1024,
		ExtendedTags: []string{},
	}

	tapirMsg := tapir.TapirMsg{
		SrcName:   "dns-tapir",
		Creator:   "",
		MsgType:   "observation",
		ListType:  "doubtlist",
		Added:     []tapir.Domain{domain},
		Removed:   []tapir.Domain{},
		Msg:       "",
		TimeStamp: time.Now(),
		TimeStr:   "",
	}

	outMsg, err := json.Marshal(tapirMsg)
	if err != nil {
		h.Log.Error("Error serializing message, discarding...")
		return "", err
	}

	return string(outMsg), nil
}

func (h Handle) ExtractDomain(msgJson string) (string, error) {
    var newQnameEvent protocols.EventsMqttMessageNewQnameJson
    dec := json.NewDecoder(strings.NewReader(msgJson))

	dec.DisallowUnknownFields()

    err := dec.Decode(&newQnameEvent)
    if err != nil {
        h.Log.Error("Error decoding qname from 'new qname event' msg")
        return "", err
    }

    return string(newQnameEvent.Qname), nil
}
