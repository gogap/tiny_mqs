package mqs

import (
	"github.com/gogap/errors"
)

const (
	TINY_MQS_ERR_NS = "MQS"
)

var (
	ERR_DECODE_BODY_FAILED            = errors.TN(TINY_MQS_ERR_NS, 1, "decode body failed, {{.err}}, body: \"{{.body}}\"")
	ERR_GET_BODY_DECODE_ELEMENT_ERROR = errors.TN(TINY_MQS_ERR_NS, 2, "get body decode element error, local: {{.local}}, error: {{.err}}")
	ERR_QUEUE_NOT_EXIST               = errors.TN(TINY_MQS_ERR_NS, 3, "queue not exist, queue: {{.queueName}}")

	ERR_NO_MESSAGE = errors.TN(TINY_MQS_ERR_NS, 4, "message not exist, queue: {{.queueName}}")

	ERR_OWNER_ID_NOT_EXIST   = errors.TN(TINY_MQS_ERR_NS, 5, "owner id not exist, owner id: {{.ownerId}}")
	ERR_OWNER_ID_NOT_ALLOWED = errors.TN(TINY_MQS_ERR_NS, 6, "AccessDenied, message: The OwnerId that your Access Key Id associated to is forbidden for this operation. ownerId: {{.ownerId}}")
	ERR_INTERNAL_ERROR       = errors.TN(TINY_MQS_ERR_NS, 7, "Internal error, error: {{.err}}")

	ERR_RECEIPT_HANDLE_ERROR = errors.TN(TINY_MQS_ERR_NS, 8, "The receipt handle you provide is not valid. {{.handle}}")

	ERR_OWNER_ID_IS_EMPTY   = errors.TN(TINY_MQS_ERR_NS, 9, "owner id is empty")
	ERR_QUEUE_NAME_IS_EMPTY = errors.TN(TINY_MQS_ERR_NS, 10, "queue name is empty")
)
