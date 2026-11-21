package tls

import (
	"bubble/internal/crypto"
	"fmt"
	"time"
)

var (
	DecryptErr     = fmt.Errorf("Decrypt error")
	SendTimeOLdErr = fmt.Errorf("Send time old")
	ParseTimeErr   = fmt.Errorf("Cant parse send time")
	NonceUseTwice  = fmt.Errorf("Nonce used twice")
)

type TLSData struct {
	Device      string
	Login       string
	Password    string
	SendTime    string
	Nonce       string
	RequestBody []byte
}

const maxRequestLatency = 30 * time.Second

func QueryCheck(nonces map[string]string, req *TLSData, key string) (err error) {
	req.SendTime, err = crypto.StringDecrypt(req.SendTime, key)
	if err != nil {
		return DecryptErr
	}
	if isOld, err := checkRequestLatencyOld(req.SendTime); isOld || err != nil {
		if isOld {
			return SendTimeOLdErr
		}

		if err != nil {
			return ParseTimeErr
		}
	}

	req.Nonce, err = crypto.StringDecrypt(req.Nonce, key)
	if err != nil {
		return DecryptErr
	}

	if nonce, ok := nonces[req.Login]; ok {
		if req.Nonce == nonce {
			return NonceUseTwice
		}
	}

	nonces[req.Login] = req.Nonce

	return nil
}

func checkRequestLatencyOld(sendTime string) (latencyOld bool, err error) {
	latencyOld = true

	var reqSendTime time.Time
	reqSendTime, err = time.Parse(time.RFC3339, sendTime)
	if err != nil {
		return
	}

	if !reqSendTime.Add(maxRequestLatency).Before(time.Now()) {
		latencyOld = false
	}
	return
}
