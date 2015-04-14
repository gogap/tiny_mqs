package mqs

import (
	"time"
)

type QueueProperty struct {
	Name                   string
	MaxMessageSize         int32
	MessageRetentionPeriod time.Duration
	VisibilityTimeout      time.Duration
	PollingWaitSeconds     time.Duration
}

func DefaultQueueProperty(name string) QueueProperty {
	return QueueProperty{
		Name:                   name,
		MaxMessageSize:         65535,
		MessageRetentionPeriod: 70 * time.Second,
		VisibilityTimeout:      60 * time.Second,
		PollingWaitSeconds:     30 * time.Second,
	}
}
