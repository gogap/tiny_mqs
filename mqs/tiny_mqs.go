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
	mqsPools    map[string]*MessagePool //key: host
	hostOwnerId map[string]string

	conf TinyMQSConfig
}

func NewTinyMQS() *TinyMQS {
	return &TinyMQS{
		mqsPools:    make(map[string]*MessagePool),
		hostOwnerId: make(map[string]string),
	}
}

func (p *TinyMQS) CreatePools(pools []PoolConfig) (err error) {
	for _, pool := range pools {
		if pool.OwnerId == "" {
			err = ErrOwnerIDIsEmpty.New()
			return
		}

		if pool.Host == "" {
			err = ErrHostIsEmpty.New(errors.Params{"ownerId": pool.OwnerId})
			return
		}

		if _, exist := p.mqsPools[pool.Host]; exist {
			err = ErrPoolAlreadyExist.New(errors.Params{"ownerId": pool.OwnerId})
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
					err = ErrQueueNameIsEmpty.New(errors.Params{"ownerId": pool.OwnerId})
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
		p.mqsPools[pool.Host] = NewMessagePool(mode, properties...)
		p.hostOwnerId[pool.Host] = pool.OwnerId
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

	if ErrNoMessage.IsEqual(err) {
		errMsg = ErrorMessageResponse{
			Code:      "MessageNotExist",
			Message:   "Message not exist.",
			RequestId: requestId,
			HostId:    hostId,
		}
		statusCode = http.StatusNotFound
	} else if ErrPoolNotExist.IsEqual(err) {
		errMsg = ErrorMessageResponse{
			Code:      "AccessDenied",
			Message:   "The OwnerId that your Access Key Id associated to is forbidden for this operation.",
			RequestId: requestId,
			HostId:    hostId,
		}
		statusCode = http.StatusForbidden
	} else if ErrBadReceiptHandle.IsEqual(err) {
		errMsg = ErrorMessageResponse{
			Code:      "ReceiptHandleError",
			Message:   "The receipt handle you provide is not valid.",
			RequestId: requestId,
			HostId:    hostId,
		}
		statusCode = http.StatusForbidden
	} else if ErrQueueNotExist.IsEqual(err) {
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

	if pool, exist := p.mqsPools[trimHost(req.Host)]; !exist {
		err = ErrPoolNotExist.New()
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
		strNumOfMessages := req.FormValue("numOfMessages")

		isMulti := false
		if strNumOfMessages != "" {
			isMulti = true
		}

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
				if queueMsg, e := p.receive(pool, queueName); e != nil {
					if !ErrNoMessage.IsEqual(e) {
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
					errChan <- ErrNoMessage.New(errors.Params{"queueName": queueName})
					return
				}
			}
		}(errorChan, messageChan, queueName, waitSecond)

		select {
		case msg := <-messageChan:
			{
				// TODO: implament true batch receive
				if !isMulti {
					if xmlData, e := xml.Marshal(&msg); e != nil {
						err = ErrInternalError.New(errors.Params{"err": e})
					} else {
						ownerId := p.hostOwnerId[trimHost(req.Host)]
						resp.Write([]byte(xml.Header + string(xmlData)))
						fmt.Printf("[*]ReceiveMessage:\n\tOwnerId:%s\n\tQueue:%s\n\t%s\n\n", ownerId, queueName, string(xmlData))
					}
				} else {
					messages := BatchMessageReceiveResponse{Messages: []MessageReceiveResponse{msg}}
					if xmlData, e := xml.Marshal(&messages); e != nil {
						err = ErrInternalError.New(errors.Params{"err": e})
					} else {
						ownerId := p.hostOwnerId[trimHost(req.Host)]
						resp.Write([]byte(xml.Header + string(xmlData)))
						fmt.Printf("[*]ReceiveMessages:\n\tOwnerId:%s\n\tQueue:%s\n\t%s\n\n", ownerId, queueName, string(xmlData))
					}
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

	pool, exist := p.mqsPools[trimHost(req.Host)]
	if !exist {
		err = ErrPoolNotExist.New()
		return
	}

	queueName := params["_1"]
	reqMsg := MessageSendRequest{}

	var reqBody []byte
	if reqBody, err = ioutil.ReadAll(req.Body); err != nil {
		return
	}

	if err = xml.Unmarshal(reqBody, &reqMsg); err != nil {
		return
	}

	msgId := ""
	if msgId, err = p.send(pool, queueName, time.Duration(reqMsg.DelaySeconds)*time.Second, reqMsg.Priority, reqMsg.MessageBody); err != nil {
		return
	}

	ownerId := p.hostOwnerId[trimHost(req.Host)]

	fmt.Printf("[*]SendMessage:\n\tOwnerId:%s\n\tQueue:%s\n\t%s\n\n", ownerId, queueName, string(reqBody))

	respMsg := MessageSendResponse{
		MessageId:      msgId,
		MessageBodyMD5: fmt.Sprintf("%0x", md5.Sum(reqBody)),
	}

	if xmlData, e := xml.Marshal(&respMsg); e != nil {
		err = ErrInternalError.New(errors.Params{"err": e})
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

	pool, exist := p.mqsPools[trimHost(req.Host)]
	if !exist {
		err = ErrPoolNotExist.New()
		return
	}

	queueName := params["_1"]

	strReceiptHandle := req.FormValue("ReceiptHandle")
	if strReceiptHandle == "" {
		ErrBadReceiptHandle.New(errors.Params{"handle": strReceiptHandle})
		return
	}

	if err = p.delete(pool, queueName, strReceiptHandle); err != nil {
		return
	}

	resp.WriteHeader(http.StatusNoContent)
}

func (p *TinyMQS) receive(pool *MessagePool, queueName string) (message TinyQueueMessage, err error) {
	message, err = pool.Dequeue(queueName)
	return
}

func (p *TinyMQS) send(pool *MessagePool, queueName string, delaySeconds time.Duration, priority int64, body Base64Bytes) (id string, err error) {
	id, err = pool.Enqueue(queueName, delaySeconds, priority, body)
	return
}

func (p *TinyMQS) delete(pool *MessagePool, queueName, receiptHandle string) (err error) {
	err = pool.Delete(queueName, receiptHandle)
	return
}

func trimHost(host string) string {
	portIndex := strings.Index(host, ":")
	return host[0:portIndex]
}
