package mqs

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMessagePool(t *testing.T) {

	Convey("TestEnqueue", t, func() {

		Convey("DefaultMode", func() {
			pool := NewMessagePool(PoolModeDefault, QueueProperty{
				Name:                   "test",
				MaxMessageSize:         65535,
				MessageRetentionPeriod: 70 * time.Second,
				VisibilityTimeout:      60 * time.Second,
				PollingWaitSeconds:     30 * time.Second,
			})

			_, err := pool.Enqueue("test", 8, 8, Base64Bytes("message"))

			ShouldBeNil(err)

			queues := pool.queueMessageIndex["test"]

			ShouldEqual(len(queues), 1)

			msgId := queues[0]
			ShouldEqual(pool.queueMessages[msgId].Body, Base64Bytes("message"))

		})

	})

	Convey("TestDequeue", t, func() {

		Convey("QueueExsist", func() {
			pool := NewMessagePool(PoolModeDefault, QueueProperty{
				Name:                   "test",
				MaxMessageSize:         65535,
				MessageRetentionPeriod: 70 * time.Second,
				VisibilityTimeout:      60 * time.Second,
				PollingWaitSeconds:     30 * time.Second,
			})

			_, err := pool.Enqueue("test", 8, Base64Bytes("message"))

			ShouldBeNil(err)

			queues := pool.queueMessageIndex["test"]

			ShouldEqual(len(queues), 1)

			msgId := queues[0]
			ShouldEqual(pool.queueMessages[msgId].Body, Base64Bytes("message"))

			msg, e := pool.Dequeue("test")

			ShouldBeNil(e)

			ShouldEqual(msg.Body, Base64Bytes("message"))

			_, e = pool.Dequeue("test")

			ShouldNotBeNil(e)

		})

		Convey("QueueNotExistAndIsDefualtMode", func() {
			pool := NewMessagePool(PoolModeDefault)

			_, err := pool.Enqueue("test", 8, Base64Bytes("message"))

			ShouldNotBeNil(err)
		})

		Convey("QueueNotExistAndIsAutoCreateMode", func() {
			pool := NewMessagePool(PoolModeAutoCreateQueue)

			_, err := pool.Enqueue("test", 8, Base64Bytes("message"))

			ShouldBeNil(err)

			queues := pool.queueMessageIndex["test"]

			ShouldEqual(len(queues), 1)

			msgId := queues[0]
			ShouldEqual(pool.queueMessages[msgId].Body, Base64Bytes("message"))

			msg, e := pool.Dequeue("test")

			ShouldBeNil(e)

			ShouldEqual(msg.Body, Base64Bytes("message"))

			_, e = pool.Dequeue("test")

			ShouldNotBeNil(e)

		})

	})

}
