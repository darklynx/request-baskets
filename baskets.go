package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const toMs = int64(time.Millisecond) / int64(time.Nanosecond)

// DoNotForwardHeader indicates whether request can (0) or cannot (1) be forwarded
const DoNotForwardHeader = "X-Do-Not-Forward"

// BasketConfig describes single basket configuration.
type BasketConfig struct {
	ForwardURL    string `json:"forward_url"`
	ProxyResponse bool   `json:"proxy_response"`
	InsecureTLS   bool   `json:"insecure_tls"`
	ExpandPath    bool   `json:"expand_path"`
	Capacity      int    `json:"capacity"`
}

// ResponseConfig describes response that is generates by service upon HTTP request sent to a basket.
type ResponseConfig struct {
	Status     int         `json:"status"`
	Headers    http.Header `json:"headers"`
	Body       string      `json:"body"`
	IsTemplate bool        `json:"is_template"`
}

// BasketAuth describes basket authentication response that is sent when new basket is created.
type BasketAuth struct {
	Token string `json:"token"`
}

// RequestData describes collected request data.
type RequestData struct {
	Date          int64       `json:"date"`
	Header        http.Header `json:"headers"`
	ContentLength int64       `json:"content_length"`
	Body          string      `json:"body"`
	Method        string      `json:"method"`
	Path          string      `json:"path"`
	Query         string      `json:"query"`
}

// RequestsPage describes a page with collected requests.
type RequestsPage struct {
	Requests   []*RequestData `json:"requests"`
	Count      int            `json:"count"`
	TotalCount int            `json:"total_count"`
	HasMore    bool           `json:"has_more"`
}

// RequestsQueryPage describes a page of found requests if search filter is applied.
type RequestsQueryPage struct {
	Requests []*RequestData `json:"requests"`
	HasMore  bool           `json:"has_more"`
}

// BasketNamesPage describes a page with basket names managed by service.
type BasketNamesPage struct {
	Names   []string `json:"names"`
	Count   int      `json:"count"`
	HasMore bool     `json:"has_more"`
}

// BasketNamesQueryPage describes a page with found basket names if search filter is applied.
type BasketNamesQueryPage struct {
	Names   []string `json:"names"`
	HasMore bool     `json:"has_more"`
}

// Basket is an interface that represent request basket entity to collects HTTP requests
type Basket interface {
	Config() BasketConfig
	Update(config BasketConfig)
	Authorize(token string) bool

	GetResponse(method string) *ResponseConfig
	SetResponse(method string, response ResponseConfig)

	Add(req *http.Request) *RequestData
	Clear()

	Size() int
	GetRequests(max int, skip int) RequestsPage
	FindRequests(query string, in string, max int, skip int) RequestsQueryPage
}

// BasketsDatabase is an interface that represent database to manage collection of request baskets
type BasketsDatabase interface {
	Create(name string, config BasketConfig) (BasketAuth, error)
	Get(name string) Basket
	Delete(name string)

	Size() int
	GetNames(max int, skip int) BasketNamesPage
	FindNames(query string, max int, skip int) BasketNamesQueryPage

	Release()
}

// ToRequestData converts HTTP Request object into RequestData holder
func ToRequestData(req *http.Request) *RequestData {
	data := new(RequestData)

	data.Date = time.Now().UnixNano() / toMs
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
func (req *RequestData) Forward(client *http.Client, config BasketConfig, basket string) (*http.Response, error) {
	forwardURL, err := url.ParseRequestURI(config.ForwardURL)
	if err != nil {
		return nil, fmt.Errorf("Invalid forward URL: %s - %s", config.ForwardURL, err)
	}

	// expand path
	if config.ExpandPath && len(req.Path) > len(basket)+1 {
		forwardURL.Path = expandURL(forwardURL.Path, req.Path, basket)
	}

	// append query
	if len(req.Query) > 0 {
		if len(forwardURL.RawQuery) > 0 {
			forwardURL.RawQuery += "&" + req.Query
		} else {
			forwardURL.RawQuery = req.Query
		}
	}

	forwardReq, err := http.NewRequest(req.Method, forwardURL.String(), strings.NewReader(req.Body))
	if err != nil {
		return nil, fmt.Errorf("Failed to create forward request: %s", err)
	}

	// copy headers
	for header, vals := range req.Header {
		for _, val := range vals {
			forwardReq.Header.Add(header, val)
		}
	}
	// headers cleanup
	forwardHeadersCleanup(forwardReq)
	// set do not forward header
	forwardReq.Header.Set(DoNotForwardHeader, "1")

	// forward request
	response, err := client.Do(forwardReq)
	if err != nil {
		// HTTP issue during forwarding - HTTP 502 Bad Gateway
		log.Printf("[warn] failed to forward request for basket: %s - %s", basket, err)
		badGatewayResp := &http.Response{
			StatusCode: http.StatusBadGateway,
			Header:     http.Header{},
			Body:       ioutil.NopCloser(strings.NewReader(fmt.Sprintf("Failed to forward request: %s", err)))}
		badGatewayResp.Header.Set("Content-Type", "text/plain")

		return badGatewayResp, nil
	}

	return response, nil
}

// forwardHeadersCleanup removes headers that may corrupt the underlying connection when forwarding request
func forwardHeadersCleanup(req *http.Request) {
	// Must not be used in HTTP/2
	req.Header.Del("Connection")
	req.Header.Del("Upgrade")
	req.Header.Del("TE") // only "trailers" supported in HTTP/2

	// TODO: find out what else may break or corrupt the forwarding
}

func expandURL(url string, original string, basket string) string {
	return strings.TrimSuffix(url, "/") + strings.TrimPrefix(original, "/"+basket)
}

// Matches checks if RequestData matches the search criterea.
func (req *RequestData) Matches(query string, in string) bool {
	// detect where to search
	inBody := false
	inQuery := false
	inHeaders := false
	switch in {
	case "body":
		inBody = true
	case "query":
		inQuery = true
	case "headers":
		inHeaders = true
	default:
		inBody = true
		inQuery = true
		inHeaders = true
	}

	if inBody && strings.Contains(req.Body, query) {
		return true
	}

	if inQuery && strings.Contains(req.Query, query) {
		return true
	}

	if inHeaders {
		for _, vals := range req.Header {
			for _, val := range vals {
				if strings.Contains(val, query) {
					return true
				}
			}
		}
	}

	return false
}
