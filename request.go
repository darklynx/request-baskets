package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
)

type Request struct {
	Header        http.Header `json:"headers"`
	ContentLength int64       `json:"content_length"`
	Body          string      `json:"body"`
	Method        string      `json:"method"`
	Path          string      `json:"path"`
	Query         string      `json:"query"`
}

type Requests struct {
	Requests []*Request `json:"requests"`
	Count    int        `json:"count"`
	Size     int        `json:"size"`
	HasMore  bool       `json:"has_more"`
}

type RequestDb struct {
	sync.RWMutex
	requests []*Request
	capacity int
	Count    int
}

func makeRequest(r *http.Request) *Request {
	req := new(Request)

	req.Header = make(http.Header)
	for k, v := range r.Header {
		req.Header[k] = v
	}

	req.ContentLength = r.ContentLength
	req.Method = r.Method
	req.Path = r.URL.Path
	req.Query = r.URL.RawQuery

	body, _ := ioutil.ReadAll(r.Body)
	req.Body = string(body)

	return req
}

func (db *RequestDb) applyLimit() {
	// Keep requests up to specified capacity
	if len(db.requests) > db.capacity {
		db.requests = db.requests[:db.capacity]
	}
}

func (db *RequestDb) Add(r *http.Request) *Request {
	db.Lock()
	defer db.Unlock()

	req := makeRequest(r)
	db.requests = append([]*Request{req}, db.requests...)
	// Keep number of all collected requests
	db.Count++

	db.applyLimit()

	return req
}

func (db *RequestDb) Clear() {
	db.Lock()
	defer db.Unlock()

	// Reset requests and counter
	db.requests = make([]*Request, 0, db.capacity)
	db.Count = 0
}

func (db *RequestDb) UpdateCapacity(capacity int) {
	if capacity != db.capacity {
		db.Lock()
		defer db.Unlock()

		db.capacity = capacity
		db.applyLimit()
	}
}

func (db *RequestDb) ToJson(max int, skip int) ([]byte, error) {
	db.RLock()
	defer db.RUnlock()

	size := len(db.requests)
	last := skip + max

	reqs := Requests{
		Count:   db.Count,
		Size:    size,
		HasMore: last < size}

	if skip < size {
		if last > size {
			last = size
		}
		reqs.Requests = db.requests[skip:last]
	}

	return json.Marshal(reqs)
}

func MakeRequestDb(capacity int) *RequestDb {
	return &RequestDb{
		requests: make([]*Request, 0, capacity),
		capacity: capacity,
		Count:    0}
}
