package msgutil

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
)

type Slack struct {
	MySigningSecret string
	LogWriter       io.Writer
	Payload
}

type Payload struct {
	Token       string `payload:"token"`
	Command     string `payload:"command"`
	Text        string `payload:"text"`
	ResponseUrl string `payload:"response_url"`
	TriggerId   string `payload:"trigger_id"`
	UserId      string `payload:"user_id"`
	UserName    string `payload:"user_name"`
	TeamId      string `payload:"team_id"`
	TeamDomain  string `payload:"team_domain"`
	ChannelName string `payload:"channel_name"`
}

// Verify return error if siging secret not matched
func (s *Slack) Verify(req *http.Request) error {
	if s.MySigningSecret == "" {
		return errors.New("My Signing Secret cannot be nil")
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if s.LogWriter != nil {
		s.LogWriter.Write([]byte("=======Header=====\n"))
		for k, v := range req.Header {
			s.LogWriter.Write([]byte(fmt.Sprintf("%s:%s\n", k, v)))
		}
		s.LogWriter.Write([]byte("=======Payload=====\n"))
		s.LogWriter.Write(body)
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	requestTimestamp := req.Header.Get("X-Slack-Request-Timestamp")
	slackSignature := req.Header.Get("X-Slack-Signature")
	needToVerifyStr := fmt.Sprintf("v0:%s:%s", requestTimestamp, string(body))
	return s.compareHash(needToVerifyStr, slackSignature)
}

func (s *Slack) compareHash(needToVerifyStr string, slackSignature string) error {
	key := []byte(s.MySigningSecret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(needToVerifyStr))
	hash := []byte(h.Sum(nil))
	dst := make([]byte, hex.EncodedLen(len(hash)))
	hex.Encode(dst, hash)
	if mySignature := fmt.Sprintf("v0=%s", string(dst)); mySignature != slackSignature {
		return errors.New("Cannot verify")
	}
	return nil
}

// ParseCmd verify incoming request and populates payload to slack.Payload
func (s *Slack) ParseCmd(req *http.Request) error {
	err := s.Verify(req)
	if err != nil {
		return err
	}
	err = s.parsePayload(req)
	if err != nil {
		return err
	}
	return nil

}

func (s *Slack) parsePayload(req *http.Request) error {
	req.ParseForm()
	v := reflect.ValueOf(&s.Payload).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fv := v.Field(i)
		if f.Tag != "" {
			key := f.Tag.Get("payload")
			fv.SetString(req.Form.Get(key))
		}
	}
	return nil
}

// Response struct contains reply message
type Response struct {
	ResponseType string `json:"response_type"`
	Text         string `json:"text"`
}

// ReplyJson return json buffer from Response struct
func (s *Slack) ReplyJson(respType string, msg string) (*bytes.Buffer, error) {
	repl := Response{
		ResponseType: respType,
		Text:         msg,
	}
	byteArr, err := json.Marshal(repl)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(byteArr), nil
}

func (s *Slack) ReplyLater(respType string, msg string) error {
	jsonBuf, err := s.ReplyJson(respType, msg)
	if err != nil {
		return err
	}
	_, err = http.Post(s.ResponseUrl, "application/json", jsonBuf)
	if err != nil {
		return err
	}
	return nil
}
