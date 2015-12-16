package mqs

import (
	"encoding/base64"
	"encoding/xml"

	"github.com/gogap/errors"
)

type MessageResponse struct {
	XMLName   xml.Name `xml:"Message" json:"-"`
	Code      string   `xml:"Code,omitempty" json:"code,omitempty"`
	Message   string   `xml:"Message,omitempty" json:"message,omitempty"`
	RequestId string   `xml:"RequestId,omitempty" json:"request_id,omitempty"`
	HostId    string   `xml:"HostId,omitempty" json:"host_id,omitempty"`
}

type ErrorMessageResponse struct {
	XMLName   xml.Name `xml:"Error" json:"-"`
	Code      string   `xml:"Code,omitempty" json:"code,omitempty"`
	Message   string   `xml:"Message,omitempty" json:"message,omitempty"`
	RequestId string   `xml:"RequestId,omitempty" json:"request_id,omitempty"`
	HostId    string   `xml:"HostId,omitempty" json:"host_id,omitempty"`
}

type MessageSendRequest struct {
	XMLName      xml.Name    `xml:"Message"`
	MessageBody  Base64Bytes `xml:"MessageBody"`
	DelaySeconds int64       `xml:"DelaySeconds"`
	Priority     int64       `xml:"Priority"`
}

type ReceiptHandles struct {
	XMLName        xml.Name `xml:"ReceiptHandles"`
	ReceiptHandles []string `xml:"ReceiptHandle"`
}

type MessageSendResponse struct {
	MessageResponse
	MessageId      string `xml:"MessageId" json:"message_id"`
	MessageBodyMD5 string `xml:"MessageBodyMD5" json:"message_body_md5"`
}

type CreateQueueRequest struct {
	XMLName                xml.Name `xml:"Queue" json:"-"`
	DelaySeconds           int32    `xml:"DelaySenconds,omitempty" json:"delay_senconds,omitempty"`
	MaxMessageSize         int32    `xml:"MaximumMessageSize,omitempty" json:"maximum_message_size,omitempty"`
	MessageRetentionPeriod int32    `xml:"MessageRetentionPeriod,omitempty" json:"message_retention_period,omitempty"`
	VisibilityTimeout      int32    `xml:"VisibilityTimeout,omitempty" json:"visibility_timeout,omitempty"`
	PollingWaitSeconds     int32    `xml:"PollingWaitSeconds,omitempty" json:"polling_wait_secods,omitempty"`
}

type MessageReceiveResponse struct {
	MessageResponse
	ReceiptHandle    string      `xml:"ReceiptHandle" json:"receipt_handle"`
	MessageBodyMD5   string      `xml:"MessageBodyMD5" json:"message_body_md5"`
	MessageBody      Base64Bytes `xml:"MessageBody" json:"message_body"`
	EnqueueTime      int64       `xml:"EnqueueTime" json:"enqueue_time"`
	NextVisibleTime  int64       `xml:"NextVisibleTime" json:"next_visible_time"`
	FirstDequeueTime int64       `xml:"FirstDequeueTime" json:"first_dequeue_time"`
	DequeueCount     int64       `xml:"DequeueCount" json:"dequeue_count"`
	Priority         int64       `xml:"Priority" json:"priority"`
}

type MessageVisibilityChangeResponse struct {
	XMLName         xml.Name `xml:"ChangeVisibility" json:"-"`
	ReceiptHandle   string   `xml:"ReceiptHandle" json:"receipt_handle"`
	NextVisibleTime int64    `xml:"NextVisibleTime" json:"next_visible_time"`
}

type Queue struct {
	QueueURL string `xml:"QueueURL" json:"url"`
}

type Queues struct {
	XMLName    xml.Name    `xml:"Queues" json:"-"`
	Queues     []Queue     `xml:"Queue" json:"queues"`
	NextMarker Base64Bytes `xml:"NextMarker" json:"next_marker"`
}

type BatchMessageSendRequest struct {
	XMLName  xml.Name             `xml:"Messages"`
	Messages []MessageSendRequest `xml:"Message"`
}

type BatchMessageReceiveResponse struct {
	XMLName  xml.Name                 `xml:"Messages" json:"-"`
	Messages []MessageReceiveResponse `xml:"Message" json:"messages"`
}

type Base64Bytes []byte

func (p Base64Bytes) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeElement(base64.StdEncoding.EncodeToString(p), start)
	return nil
}

func (p *Base64Bytes) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	var content string
	if e := d.DecodeElement(&content, &start); e != nil {
		err = ErrGetBodyDecodeElementError.New(errors.Params{"err": e, "local": start.Name.Local})
		return
	}

	buf := make([]byte, len(content))
	if length, e := base64.StdEncoding.Decode(buf, []byte(content)); e != nil {
		err = ErrDecodeBodyFailed.New(errors.Params{"err": e, "body": content})
		return
	} else {
		*p = Base64Bytes(buf[0:length])
	}

	return nil
}
