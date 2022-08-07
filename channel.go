package mediasoup

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"github.com/jiyeyuran/mediasoup-go/netcodec"
)

type Channel struct {
	IEventEmitter
	logger   logr.Logger
	codec    netcodec.Codec
	closed   int32
	pid      int
	nextId   int64
	sents    sync.Map
	sentChan chan sentInfo
	closeCh  chan struct{}
}

func newChannel(codec netcodec.Codec, pid int) *Channel {
	logger := NewLogger("Channel")

	logger.V(1).Info("constructor()")

	channel := &Channel{
		IEventEmitter: NewEventEmitter(),
		logger:        logger,
		codec:         codec,
		pid:           pid,
		sentChan:      make(chan sentInfo),
		closeCh:       make(chan struct{}),
	}

	return channel
}

func (c *Channel) Start() {
	go c.runWriteLoop()
	go c.runReadLoop()
}

func (c *Channel) Close() error {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.logger.V(1).Info("close()")
		close(c.closeCh)
		return c.codec.Close()
	}
	return nil
}

func (c *Channel) Closed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

func (c *Channel) Request(method string, internal internalData, data ...interface{}) (rsp workerResponse) {
	if c.Closed() {
		rsp.err = NewInvalidStateError("PayloadChannel closed")
		return
	}
	id := atomic.AddInt64(&c.nextId, 1)
	atomic.CompareAndSwapInt64(&c.nextId, 4294967295, 1)

	c.logger.V(1).Info("request()", "method", method, "id", id)

	request := workerRequest{
		Id:       id,
		Method:   method,
		Internal: internal,
	}
	if len(data) > 0 {
		request.Data = data[0]
	}
	requestData, _ := json.Marshal(request)

	if len(requestData) > NS_MESSAGE_MAX_LEN {
		return workerResponse{err: errors.New("Channel request too big")}
	}

	sent := sentInfo{
		method:      method,
		requestData: requestData,
		respCh:      make(chan workerResponse),
	}
	c.sents.Store(id, sent)
	defer c.sents.Delete(id)

	timer := time.NewTimer(time.Duration(3000) * time.Millisecond)
	defer timer.Stop()

	// send request
	select {
	case c.sentChan <- sent:
	case <-timer.C:
		rsp.err = fmt.Errorf("Channel request timeout, id: %d, method: %s", id, method)
	case <-c.closeCh:
		rsp.err = NewInvalidStateError("Channel closed, id: %d, method: %s", id, method)
	}
	if rsp.err != nil {
		return
	}

	// wait response
	select {
	case rsp = <-sent.respCh:
	case <-timer.C:
		rsp.err = fmt.Errorf("Channel response timeout, id: %d, method: %s", id, method)
	case <-c.closeCh:
		rsp.err = NewInvalidStateError("Channel closed, id: %d, method: %s", id, method)
	}
	return
}

func (c *Channel) runWriteLoop() {
	defer c.Close()

	for {
		select {
		case sentInfo := <-c.sentChan:
			if err := c.codec.WritePayload(sentInfo.requestData); err != nil {
				sentInfo.respCh <- workerResponse{err: err}
				break
			}
		case <-c.closeCh:
			return
		}
	}
}

func (c *Channel) runReadLoop() {
	defer c.Close()

	for {
		payload, err := c.codec.ReadPayload()
		if err != nil {
			c.logger.Error(err, "read failed")
			break
		}
		c.processPayload(payload)
	}
}

func (c *Channel) processPayload(nsPayload []byte) {
	switch nsPayload[0] {
	case '{':
		c.processMessage(nsPayload)
	case 'D':
		c.logger.V(1).Info(string(nsPayload[1:]), "pid", c.pid)
	case 'W':
		c.logger.Info(string(nsPayload[1:]), "pid", c.pid, "warn", true)
	case 'E':
		c.logger.Error(nil, string(nsPayload[1:]), "pid", c.pid)
	case 'X':
		fmt.Printf("%s\n", nsPayload[1:])
	default:
		c.logger.Error(errors.New(string(nsPayload[1:])), "unexpected data", "pid", c.pid)
	}
}

func (c *Channel) processMessage(nsPayload []byte) {
	var msg struct {
		// response
		Id       int64  `json:"id,omitempty"`
		Accepted bool   `json:"accepted,omitempty"`
		Error    string `json:"error,omitempty"`
		Reason   string `json:"reason,omitempty"`
		// notification
		TargetId interface{} `json:"targetId,omitempty"`
		Event    string      `json:"event,omitempty"`
		// common data
		Data json.RawMessage `json:"data,omitempty"`
	}
	if err := json.Unmarshal(nsPayload, &msg); err != nil {
		c.logger.Error(err, "received response, failed to unmarshal to json", "payload", json.RawMessage(nsPayload))
		return
	}

	if msg.Id > 0 {
		value, ok := c.sents.Load(msg.Id)
		if !ok {
			c.logger.Error(nil, "received response does not match any sent request", "id", msg.Id)
			return
		}
		sent := value.(sentInfo)

		if msg.Accepted {
			c.logger.V(1).Info("request succeeded", "method", sent.method, "id", msg.Id)

			sent.respCh <- workerResponse{data: msg.Data}
		} else if len(msg.Error) > 0 {
			c.logger.Error(errors.New(msg.Reason), "request failed", "method", sent.method, "id", msg.Id)

			if msg.Error == "TypeError" {
				sent.respCh <- workerResponse{err: NewTypeError(msg.Reason)}
			} else {
				sent.respCh <- workerResponse{err: errors.New(msg.Reason)}
			}
		} else {
			c.logger.Error(nil, "received response is not accepted nor rejected", "method", sent.method, "id", msg.Id)
		}
	} else if msg.TargetId != nil && len(msg.Event) > 0 {
		var targetId string
		// The type of msg.TargetId should be string or float64
		switch v := msg.TargetId.(type) {
		case string:
			targetId = v
		case float64:
			targetId = fmt.Sprintf("%0.0f", v)
		default:
			targetId = fmt.Sprintf("%v", v)
		}
		go c.SafeEmit(targetId, msg.Event, msg.Data)
	} else {
		c.logger.Error(nil, "received message is not a response nor a notification")
	}
}
