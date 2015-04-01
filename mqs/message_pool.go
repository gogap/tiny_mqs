package mqs

import (
	"encoding/base64"
	"sync"
	"time"

	"github.com/gogap/errors"
	"github.com/nu7hatch/gouuid"
)

type MessagePoolMode int32

const (
	PoolModeDefault         MessagePoolMode = 0
	PoolModeAutoCreateQueue MessagePoolMode = 1
)

type TinyQueueMessage struct {
	Id               string
	ReceiptHandle    string
	Body             Base64Bytes
	EnqueueTime      time.Time
	DequeueCount     int64
	Priority         int64
	DelaySeconds     time.Duration
	FirstDequeueTime time.Duration
	NextDequeueTime  time.Duration
	destroyTime      time.Duration
}

func NewTinyQueueMessage(body Base64Bytes, delaySeconds time.Duration, priority int64, lifeCycle time.Duration) (message *TinyQueueMessage) {
	UUIDMsgId, _ := uuid.NewV4()
	UUIDHandler, _ := uuid.NewV4()

	var pri int64 = 8
	if priority > 0 && pri <= 16 {
		pri = priority
	}

	if lifeCycle < 60*time.Second {
		lifeCycle = 60 * time.Second
	}

	return &TinyQueueMessage{
		Id:               UUIDMsgId.String(),
		ReceiptHandle:    base64.StdEncoding.EncodeToString([]byte(UUIDHandler.String())),
		Body:             body,
		EnqueueTime:      time.Now().UTC(),
		Priority:         pri,
		DelaySeconds:     delaySeconds,
		DequeueCount:     0,
		FirstDequeueTime: 0,
		NextDequeueTime:  0,
		destroyTime:      lifeCycle,
	}
}

type MessagePool struct {
	queueMessageIndex map[string][]string
	queueMessages     map[string]*TinyQueueMessage
	poolMode          MessagePoolMode

	queuesProperty map[string]QueueProperty

	queueMsgLocker sync.Mutex
}

func NewMessagePool(poolMode MessagePoolMode, queuesProperties ...QueueProperty) *MessagePool {
	properties := make(map[string]QueueProperty)
	queueMessageIndex := make(map[string][]string)

	if queuesProperties != nil && len(queuesProperties) > 0 {
		for _, property := range queuesProperties {
			properties[property.Name] = property
			queueMessageIndex[property.Name] = []string{}
		}
	}

	return &MessagePool{
		poolMode:          poolMode,
		queuesProperty:    properties,
		queueMessageIndex: queueMessageIndex,
		queueMessages:     make(map[string]*TinyQueueMessage),
	}
}

func (p *MessagePool) Enqueue(queueName string, delaySeconds time.Duration, priority int64, body Base64Bytes) (id string, err error) {
	p.queueMsgLocker.Lock()
	defer p.queueMsgLocker.Unlock()

	queue, exist := p.queueMessageIndex[queueName]

	if !exist {
		if p.poolMode == PoolModeDefault {
			err = ERR_QUEUE_NOT_EXIST.New(errors.Params{"queueName": queueName})
			return
		} else {
			p.queueMessageIndex[queueName] = []string{}
		}
	}

	var queueProperty QueueProperty

	if property, exist := p.queuesProperty[queueName]; exist {
		queueProperty = property
	} else {
		queueProperty = DefaultQueueProperty(queueName)
	}

	newMsg := NewTinyQueueMessage(body, delaySeconds, priority, queueProperty.MessageRetentionPeriod)
	p.queueMessages[newMsg.Id] = newMsg
	p.queueMessageIndex[queueName] = append(queue, newMsg.Id)
	id = newMsg.Id

	return
}

func (p *MessagePool) Dequeue(queueName string) (message TinyQueueMessage, err error) {
	p.queueMsgLocker.Lock()
	defer p.queueMsgLocker.Unlock()

	queue, exist := p.queueMessageIndex[queueName]

	if !exist {
		if p.poolMode == PoolModeDefault {
			err = ERR_QUEUE_NOT_EXIST.New(errors.Params{"queueName": queueName})
		} else {
			err = ERR_NO_MESSAGE.New(errors.Params{"queueName": queueName})
		}
		return
	}

	if queue == nil || len(queue) == 0 {
		err = ERR_NO_MESSAGE.New(errors.Params{"queueName": queueName})
		return
	} else {
		putQueueBack := []string{}

		var queueProperty QueueProperty

		if property, exist := p.queuesProperty[queueName]; exist {
			queueProperty = property
		} else {
			queueProperty = DefaultQueueProperty(queueName)
		}

		getted := false
		for _, messageId := range queue {
			if msg, exist := p.queueMessages[messageId]; exist {
				now := time.Now().UTC()

				if now.Sub(msg.EnqueueTime) >= msg.DelaySeconds && now.Sub(msg.EnqueueTime.Add(msg.FirstDequeueTime)) >= msg.NextDequeueTime {
					if now.Sub(msg.EnqueueTime) < msg.destroyTime {
						if msg.DequeueCount == 0 {
							msg.FirstDequeueTime = now.Sub(msg.EnqueueTime)
						}

						msg.DequeueCount++
						msg.NextDequeueTime = msg.FirstDequeueTime + queueProperty.MessageRetentionPeriod
						getted = true
						message = *msg
					}
				}

				if now.Sub(msg.EnqueueTime) < msg.destroyTime {
					putQueueBack = append(putQueueBack, msg.Id)
				}
			}
		}

		if !getted {
			err = ERR_NO_MESSAGE.New(errors.Params{"queueName": queueName})
			return
		}
		return
	}
}

func (p *MessagePool) Delete(queueName string, receiptHandle string) (err error) {
	p.queueMsgLocker.Lock()
	defer p.queueMsgLocker.Unlock()

	for id, msg := range p.queueMessages {
		if msg.ReceiptHandle == receiptHandle {
			delete(p.queueMessages, id)
			return
		}
	}

	return
}

func (p *MessagePool) GetQueueProperty(queueName string) (property QueueProperty, err error) {
	if prop, exists := p.queuesProperty[queueName]; !exists {
		if p.poolMode == PoolModeDefault {
			err = ERR_QUEUE_NOT_EXIST.New(errors.Params{"queueName": queueName})
			return
		} else {
			property = DefaultQueueProperty(queueName)
			return
		}
	} else {
		property = prop
	}
	return
}
