package mediasoup

import "encoding/json"

type H map[string]interface{}

func Bool(b bool) *bool {
	return &b
}

type DumpResult struct {
	Data []byte
	Err  error
}

func NewDumpResult(data []byte, err error) DumpResult {
	return DumpResult{
		Data: data,
		Err:  err,
	}
}

func (r DumpResult) Unmarshal(v interface{}) error {
	if r.Err != nil {
		return r.Err
	}
	return json.Unmarshal(r.Data, v)
}

type RouterDump struct {
	Id                               string              `json:"id,omitempty"`
	MapProducerIdConsumerIds         map[string][]string `json:"mapProducerIdConsumerIds,omitempty"`
	MapConsumerIdProducerId          map[string]string   `json:"mapConsumerIdProducerId,omitempty"`
	MapDataProducerIdDataConsumerIds map[string][]string `json:"mapDataProducerIdDataConsumerIds,omitempty"`
	MapDataConsumerIdDataProducerId  map[string]string   `json:"mapDataConsumerIdDataProducerId,omitempty"`
	MapProducerIdObserverIds         map[string][]string `json:"mapProducerIdObserverIds,omitempty"`
	RtpObserverIds                   []string            `json:"rtpObserverIds,omitempty"`
	TransportIds                     []string            `json:"transportIds,omitempty"`
}

type TransportDump struct {
	Id              string   `json:"id,omitempty"`
	ProducerIds     []string `json:"producerIds,omitempty"`
	ConsumerIds     []string `json:"consumerIds,omitempty"`
	DataProducerIds []string `json:"dataProducerIds,omitempty"`
	DataConsumerIds []string `json:"dataConsumerIds,omitempty"`
}

type internalData struct {
	RouterId       string `json:"routerId,omitempty"`
	TransportId    string `json:"transportId,omitempty"`
	ProducerId     string `json:"producerId,omitempty"`
	ConsumerId     string `json:"consumerId,omitempty"`
	DataProducerId string `json:"dataProducerId,omitempty"`
	DataConsumerId string `json:"dataConsumerId,omitempty"`
	RtpObserverId  string `json:"rtpObserverId,omitempty"`
}
