package advancedlog

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// LogAPI is an interface for logging
type LogAPI struct {
	token  string // token of application
	url    string // url for requests
	debug  bool   // if debug is true messages won't send to the server, they will be printed by log.Println()
	client *http.Client
}

// NewLog return an exemplar of LogAPI
func NewLog(token, url string, debug bool) (*LogAPI, error) {
	l := new(LogAPI)
	if token == "" || url == "" {
		l.debug = true
	}

	l.token = token
	l.url = url
	l.debug = debug
	l.client = &http.Client{}

	return l, nil
}

// Log sent data to the server
func (l *LogAPI) Log(reqType, command, message string, reqTime int64) error {
	if reqType != "event" && reqType != "request" && reqType != "error" {
		return errors.New("Bad type")
	}

	if l.debug {
		log.Printf("Type: %s    Command: %s    Message: %s    Time: %d", reqType, command, message, reqTime)
		return nil
	}

	// /api/v1/log?token={token}&type={type}&command={command}&message={message}&time={time}
	url := "/api/v1/log?token=" + l.token + "&type=" + reqType + "&command=" + command +
		"&message=" + message + "&time=" + strconv.FormatInt(reqTime, 10)
	url = strings.Replace(url, " ", "%20", -1)

	req, _ := http.NewRequest("PUT", l.url+url, nil)
	resp, err := l.client.Do(req)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		msg, _ := ioutil.ReadAll(resp.Body)
		return errors.New(string(msg))
	}

	return nil
}
