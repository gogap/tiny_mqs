package mqs

import (
	"github.com/gogap/errors"
)

const (
	TinyMQSErrNamespace = "MQS"
)

var (
	ErrDecodeBodyFailed          = errors.TN(TinyMQSErrNamespace, 1, "decode body failed, {{.err}}, body: \"{{.body}}\"")
	ErrGetBodyDecodeElementError = errors.TN(TinyMQSErrNamespace, 2, "get body decode element error, local: {{.local}}, error: {{.err}}")
	ErrQueueNotExist             = errors.TN(TinyMQSErrNamespace, 3, "queue not exist, queue: {{.queueName}}")
	ErrNoMessage                 = errors.TN(TinyMQSErrNamespace, 4, "message not exist, queue: {{.queueName}}")
	ErrInternalError             = errors.TN(TinyMQSErrNamespace, 5, "Internal error, error: {{.err}}")
	ErrBadReceiptHandle          = errors.TN(TinyMQSErrNamespace, 6, "The receipt handle you provide is not valid. {{.handle}}")
	ErrOwnerIDIsEmpty            = errors.TN(TinyMQSErrNamespace, 7, "owner id is empty")
	ErrQueueNameIsEmpty          = errors.TN(TinyMQSErrNamespace, 8, "queue name is empty, owner: {{.ownerID}}")
	ErrHostIsEmpty               = errors.TN(TinyMQSErrNamespace, 9, "host is empty for pool owner: {{.ownerID}}")
	ErrPoolAlreadyExist          = errors.TN(TinyMQSErrNamespace, 10, "pool already exist, owner: {{.ownerID}}")
	ErrPoolNotExist              = errors.TN(TinyMQSErrNamespace, 11, "host did not have related pool")
)
