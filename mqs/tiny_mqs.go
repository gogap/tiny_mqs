package mqs

import (
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-martini/martini"
	"github.com/gogap/errors"
	"github.com/gogap/logs"
	"github.com/nu7hatch/gouuid"
)

type TinyMQS struct {
	mqsPools map[string]*MessagePool //key: ownerId

	conf TinyMQSConfig
}

func NewTinyMQS() *TinyMQS {
	return &TinyMQS{
		mqsPools: make(map[string]*MessagePool),
	}
}

func (p *TinyMQS) CreatePools(pools []PoolConfig) (err error) {
	for _, pool := range pools {
		if pool.OwnerId == "" {
			err = ERR_OWNER_ID_IS_EMPTY.New()
			return
		}

		mode := PoolModeDefault

		if pool.Mode != "" {
			if pool.Mode == "auto_create" {
				mode = PoolModeAutoCreateQueue
			}
		}

		properties := []QueueProperty{}

		if pool.Queues != nil {
			for _, property := range pool.Queues {
				if property.Name == "" {
					err = ERR_QUEUE_NAME_IS_EMPTY.New()
					return
				}

				prop := QueueProperty{
					Name:                   property.Name,
					MaxMessageSize:         property.MaxMessageSize,
					MessageRetentionPeriod: time.Duration(property.MessageRetentionPeriod) * time.Second,
					VisibilityTimeout:      time.Duration(property.VisibilityTimeout) * time.Second,
					PollingWaitSeconds:     time.Duration(property.PollingWaitSeconds) * time.Second,
				}

				properties = append(properties, prop)
			}
		}
		p.mqsPools[pool.OwnerId] = NewMessagePool(mode, properties...)
	}
	return
}

func (p *TinyMQS) Run() {
	err := p.conf.LoadConfig("conf/tiny_mqs.conf")

	if err != nil {
		panic(err)
	}
	err = p.CreatePools(p.conf.Pools)

	if err != nil {
		panic(err)
	}

	m := martini.Classic()

	m.Post("/**/messages", p.SendMessage)
	m.Get("/**/messages", p.ReceiveMessage)
	m.Delete("/**/messages", p.DeleteMessage)

	http.Handle("/", m)

	m.RunOnAddr(p.conf.HTTP.Address)
}

func getOwnerId(host string) string {
	firstPoint := strings.Index(host, ".")
	if firstPoint > 0 {
		return host[0:firstPoint]
	}
	return host
}

func getHostId(host string) string {
	return "http://" + host
}

func getRequestId() string {
	uuidReqId, _ := uuid.NewV4()
	return strings.ToUpper(strings.Replace(uuidReqId.String(), "-", "", -1))
}

func errResp(requestId string, hostId string, err error, resp http.ResponseWriter) {
	statusCode := 200
	var errMsg ErrorMessageResponse

	if ERR_NO_MESSAGE.IsEqual(err) {
		errMsg = ErrorMessageResponse{
			Code:      "MessageNotExist",
			Message:   "Message not exist.",
			RequestId: requestId,
			HostId:    hostId,
		}
		statusCode = http.StatusNotFound
	} else if ERR_OWNER_ID_NOT_ALLOWED.IsEqual(err) {
		errMsg = ErrorMessageResponse{
			Code:      "AccessDenied",
			Message:   "The OwnerId that your Access Key Id associated to is forbidden for this operation.",
			RequestId: requestId,
			HostId:    hostId,
		}
		statusCode = http.StatusForbidden
	} else if ERR_RECEIPT_HANDLE_ERROR.IsEqual(err) {
		errMsg = ErrorMessageResponse{
			Code:      "ReceiptHandleError",
			Message:   "The receipt handle you provide is not valid.",
			RequestId: requestId,
			HostId:    hostId,
		}
		statusCode = http.StatusForbidden
	} else if ERR_QUEUE_NOT_EXIST.IsEqual(err) {
		errMsg = ErrorMessageResponse{
			Code:      "QueueNotExist",
			Message:   "The queue name you provided is not exist.",
			RequestId: requestId,
			HostId:    hostId,
		}
		statusCode = http.StatusNotFound
	} else {
		errMsg = ErrorMessageResponse{
			Code:      "InternalError",
			Message:   "Internal error.",
			RequestId: requestId,
			HostId:    hostId,
		}
		statusCode = http.StatusInternalServerError
	}

	if xmlData, e := xml.Marshal(&errMsg); e != nil {
		logs.Error(e)
	} else {
		resp.WriteHeader(statusCode)
		resp.Write([]byte(xml.Header + string(xmlData)))
	}
}

func (p *TinyMQS) ReceiveMessage(resp http.ResponseWriter, req *http.Request, params martini.Params) {
	resp.Header().Set("Content-Type", "text/xml;utf-8")

	hostId := getHostId(req.Host)
	requestId := getRequestId()

	var err error
	defer func() {
		if err != nil {
			errResp(requestId, hostId, err, resp)
		}
	}()

	queueName := params["_1"]
	ownerId := getOwnerId(req.Host)

	if pool, exist := p.mqsPools[ownerId]; !exist {
		err = ERR_OWNER_ID_NOT_ALLOWED.New(errors.Params{"ownerId": ownerId})
		return
	} else {
		var queueProperty QueueProperty
		if queueProperty, err = pool.GetQueueProperty(queueName); err != nil {
			return
		}

		var waitSecond time.Duration = 0

		if queueProperty.PollingWaitSeconds > 0 {
			waitSecond = queueProperty.PollingWaitSeconds
		}

		strWaitseconds := req.FormValue("waitseconds")
		if strWaitseconds != "" {
			if sec, e := strconv.ParseInt(strWaitseconds, 10, 64); e == nil {
				if sec >= 0 && sec <= 30 {
					waitSecond = time.Duration(sec) * time.Second
				}
			} else {
				logs.Error(e)
			}
		}

		errorChan := make(chan error)
		messageChan := make(chan MessageReceiveResponse)

		defer close(errorChan)
		defer close(messageChan)

		go func(errChan chan error, msgChan chan MessageReceiveResponse, queueName string, waitSeconds time.Duration) {

			now := time.Now().UTC()
			for {
				if queueMsg, e := p.receive(ownerId, queueName); e != nil {
					if !ERR_NO_MESSAGE.IsEqual(e) {
						errChan <- e
					}
				} else {
					msgResp := MessageReceiveResponse{
						ReceiptHandle:    queueMsg.ReceiptHandle,
						MessageBodyMD5:   fmt.Sprintf("%0x", md5.Sum(queueMsg.Body)),
						MessageBody:      queueMsg.Body,
						EnqueueTime:      queueMsg.EnqueueTime.UnixNano(),
						NextVisibleTime:  queueMsg.EnqueueTime.Add(queueMsg.FirstDequeueTime).Add(queueMsg.NextDequeueTime).UnixNano(),
						FirstDequeueTime: queueMsg.EnqueueTime.Add(queueMsg.FirstDequeueTime).UnixNano(),
						DequeueCount:     queueMsg.DequeueCount,
						Priority:         queueMsg.Priority,
					}
					msgChan <- msgResp
					return
				}

				time.Sleep(time.Microsecond * 1)

				if time.Now().UTC().Sub(now) >= waitSeconds {
					errChan <- ERR_NO_MESSAGE.New(errors.Params{"queueName": queueName})
					return
				}
			}
		}(errorChan, messageChan, queueName, waitSecond)

		select {
		case msg := <-messageChan:
			{
				if xmlData, e := xml.Marshal(&msg); e != nil {
					err = ERR_INTERNAL_ERROR.New(errors.Params{"err": e})
				} else {
					resp.Write([]byte(xml.Header + string(xmlData)))
				}

				return
			}
		case e := <-errorChan:
			{
				err = e
			}
		}
	}

	return
}

func (p *TinyMQS) SendMessage(resp http.ResponseWriter, req *http.Request, params martini.Params) {
	resp.Header().Set("Content-Type", "text/xml;utf-8")

	hostId := getHostId(req.Host)
	requestId := getRequestId()

	var err error
	if err != nil {
		defer func() {
			if err != nil {
				errResp(requestId, hostId, err, resp)
			}
		}()

	}

	queueName := params["_1"]
	ownerId := getOwnerId(req.Host)

	reqMsg := MessageSendRequest{}

	var reqBody []byte
	if reqBody, err = ioutil.ReadAll(req.Body); err != nil {
		return
	}

	if err = xml.Unmarshal(reqBody, &reqMsg); err != nil {
		return
	}

	msgId := ""
	if msgId, err = p.send(ownerId, queueName, time.Duration(reqMsg.DelaySeconds)*time.Second, reqMsg.Priority, reqMsg.MessageBody); err != nil {
		return
	}

	respMsg := MessageSendResponse{
		MessageId:      msgId,
		MessageBodyMD5: fmt.Sprintf("%0x", md5.Sum(reqBody)),
	}

	if xmlData, e := xml.Marshal(&respMsg); e != nil {
		err = ERR_INTERNAL_ERROR.New(errors.Params{"err": e})
	} else {
		resp.Write([]byte(xml.Header + string(xmlData)))
	}
}

func (p *TinyMQS) DeleteMessage(resp http.ResponseWriter, req *http.Request, params martini.Params) {
	resp.Header().Set("Content-Type", "text/xml;utf-8")

	hostId := getHostId(req.Host)
	requestId := getRequestId()

	var err error
	if err != nil {
		defer func() {
			if err != nil {
				errResp(requestId, hostId, err, resp)
			}
		}()

	}

	queueName := params["_1"]
	ownerId := getOwnerId(req.Host)

	strReceiptHandle := req.FormValue("ReceiptHandle")
	if strReceiptHandle == "" {
		ERR_RECEIPT_HANDLE_ERROR.New(errors.Params{"handle": strReceiptHandle})
		return
	}

	if err = p.delete(ownerId, queueName, strReceiptHandle); err != nil {
		return
	}

	resp.WriteHeader(http.StatusNoContent)
}

func (p *TinyMQS) receive(ownerId, queueName string) (message TinyQueueMessage, err error) {
	if pool, exist := p.mqsPools[ownerId]; !exist {
		err = ERR_OWNER_ID_NOT_EXIST.New(errors.Params{"ownerId": ownerId})
		return
	} else {
		message, err = pool.Dequeue(queueName)
		return
	}
}

func (p *TinyMQS) send(ownerId, queueName string, delaySeconds time.Duration, priority int64, body Base64Bytes) (id string, err error) {
	if pool, exist := p.mqsPools[ownerId]; !exist {
		err = ERR_OWNER_ID_NOT_EXIST.New(errors.Params{"ownerId": ownerId})
		return
	} else {
		id, err = pool.Enqueue(queueName, delaySeconds, priority, body)
		return
	}
}

func (p *TinyMQS) delete(ownerId, queueName, receiptHandle string) (err error) {
	if pool, exist := p.mqsPools[ownerId]; !exist {
		err = ERR_OWNER_ID_NOT_EXIST.New(errors.Params{"ownerId": ownerId})
		return
	} else {
		err = pool.Delete(queueName, receiptHandle)
		return
	}
}
