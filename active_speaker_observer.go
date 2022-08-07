package mediasoup

import (
	"encoding/json"

	"github.com/go-logr/logr"
)

type ActiveSpeakerObserverOptions struct {
	Interval int `json:"interval"`
	AppData  H   `json:"appData,omitempty"`
}

type ActiveSpeakerObserverActivity struct {
	Producer *Producer
}

type ActiveSpeakerObserver struct {
	IRtpObserver
	logger logr.Logger
}

func newActiveSpeakerObserver(params rtpObserverParams) *ActiveSpeakerObserver {
	o := &ActiveSpeakerObserver{
		IRtpObserver: newRtpObserver(params),
		logger:       NewLogger("ActiveSpeakerObserver"),
	}
	o.handleWorkerNotifications(params)

	return o
}

func (o *ActiveSpeakerObserver) Observer() IEventEmitter {
	return o.IRtpObserver.Observer()
}

func (o *ActiveSpeakerObserver) handleWorkerNotifications(params rtpObserverParams) {
	rtpObserverId := params.internal.RtpObserverId
	getProducerById := params.getProducerById

	type eventInfo struct {
		ProducerId string `json:"producerId,omitempty"`
	}

	params.channel.On(rtpObserverId, func(event string, data []byte) {
		switch event {
		case "dominantspeaker":
			event := eventInfo{}

			if err := json.Unmarshal(data, &event); err != nil {
				o.logger.Error(err, "unmarshal events failed")
				break
			}

			dominantSpeaker := ActiveSpeakerObserverActivity{
				Producer: getProducerById(event.ProducerId),
			}
			o.SafeEmit("dominantspeaker", dominantSpeaker)

			// Emit observer event.
			o.Observer().SafeEmit("dominantspeaker", dominantSpeaker)

		default:
			o.logger.Error(nil, "ignoring unknown event", "event", event)
		}
	})
}
