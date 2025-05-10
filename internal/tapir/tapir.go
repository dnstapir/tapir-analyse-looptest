package tapir

import (
	"encoding/json"
	"time"

	"github.com/dnstapir/tapir"

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
