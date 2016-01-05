package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const TO_MS = int64(time.Millisecond) / int64(time.Nanosecond)

type BasketConfig struct {
	ForwardUrl string `json:"forward_url"`
	Capacity   int    `json:"capacity"`
}

type BasketAuth struct {
	Token string `json:"token"`
}

type RequestData struct {
	Date          int64       `json:"date"`
	Header        http.Header `json:"headers"`
	ContentLength int64       `json:"content_length"`
	Body          string      `json:"body"`
	Method        string      `json:"method"`
	Path          string      `json:"path"`
	Query         string      `json:"query"`
}

type RequestsPage struct {
	Requests   []*RequestData `json:"requests"`
	Count      int            `json:"count"`
	TotalCount int            `json:"total_count"`
	HasMore    bool           `json:"has_more"`
}

type BasketNamesPage struct {
	Names   []string `json:"names"`
	Count   int      `json:"count"`
	HasMore bool     `json:"has_more"`
}

// Basket is an interface that represent request basket entity to collects HTTP requests
type Basket interface {
	Config() BasketConfig
	Update(config BasketConfig)
	Authorize(token string) bool

	Add(req *http.Request) *RequestData
	Clear()

	Size() int
	GetRequests(max int, skip int) RequestsPage
}

// BasketsDatabase is an interface that represent database to manage collection of request baskets
type BasketsDatabase interface {
	Create(name string, config BasketConfig) (BasketAuth, error)
	Get(name string) Basket
	Delete(name string)

	Size() int
	GetNames(max int, skip int) BasketNamesPage
}

// ToRequestData converts HTTP Request object into RequestData holder
func ToRequestData(req *http.Request) *RequestData {
	data := new(RequestData)

	data.Date = time.Now().UnixNano() / TO_MS
	data.Header = make(http.Header)
	for k, v := range req.Header {
		data.Header[k] = v
	}

	data.ContentLength = req.ContentLength
	data.Method = req.Method
	data.Path = req.URL.Path
	data.Query = req.URL.RawQuery

	body, _ := ioutil.ReadAll(req.Body)
	data.Body = string(body)

	return data
}

// Forward forwards request data to specified URL
func (req *RequestData) Forward(client *http.Client, forwardUrl string) {
	body := strings.NewReader(req.Body)

	// append query
	if len(req.Query) > 0 {
		if strings.Index(forwardUrl, "?") < 0 {
			forwardUrl += "?"
		} else {
			forwardUrl += "&"
		}
		forwardUrl += req.Query
	}

	forwardReq, _ := http.NewRequest(req.Method, forwardUrl, body)

	// copy headers
	for header, vals := range req.Header {
		for _, val := range vals {
			forwardReq.Header.Add(header, val)
		}
	}

	response, _ := client.Do(forwardReq)
	io.Copy(ioutil.Discard, response.Body)
	response.Body.Close()
}
