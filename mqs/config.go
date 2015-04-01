package mqs

import (
	"encoding/json"
	"io/ioutil"
)

type HTTPConfig struct {
	Address string `json:"address"`
}

type QueueConfig struct {
	Name                   string `json:"name"`
	MaxMessageSize         int32  `json:"max_message_size"`
	MessageRetentionPeriod int64  `json:"message_retention_period"`
	VisibilityTimeout      int64  `json:"visibility_timeout"`
	PollingWaitSeconds     int64  `json:"polling_wait_seconds"`
}

type PoolConfig struct {
	OwnerId         string        `json:"owner_id"`
	AccessKeyId     string        `json:"access_key_id"`
	AccessKeySecert string        `json:"access_key_secret"`
	Mode            string        `json:"mode"`
	Queues          []QueueConfig `json:"queues"`
}

type TinyMQSConfig struct {
	HTTP  HTTPConfig   `json:"http"`
	Pools []PoolConfig `json:"pools"`
}

func (p *TinyMQSConfig) LoadConfig(filename string) (err error) {

	var data []byte
	if data, err = ioutil.ReadFile(filename); err != nil {
		return
	}

	if err = json.Unmarshal(data, p); err != nil {
		return
	}

	return
}
