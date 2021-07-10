package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestData_Forward(t *testing.T) {
	basket := "demo"

	// Test request
	data := new(RequestData)
	data.Header = make(http.Header)
	data.Header.Add("Content-Type", "application/json")
	data.Header.Add("User-Agent", "Unit-Test")
	data.Header.Add("Accept", "plain/text")
	data.Method = "POST"
	data.Body = "{ \"name\" : \"test\", \"action\" : \"add\" }"
	data.ContentLength = int64(len(data.Body))
	// path contains basket name
	data.Path = "/" + basket + "/service/actions"
	data.Query = "id=15&max=10"

	// Test HTTP server
	var forwardedData *RequestData
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedData = ToRequestData(r)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Config to forward requests to test HTTP server
	config := BasketConfig{ForwardURL: ts.URL, ExpandPath: false, Capacity: 20}
	data.Forward(new(http.Client), config, basket)

	// Validate forwarded request
	assert.Equal(t, data.Method, forwardedData.Method, "wrong request method")
	// path is not expanded during forward
	assert.Equal(t, "/", forwardedData.Path, "wrong request path")
	assert.Equal(t, data.Query, forwardedData.Query, "wrong request query")
	assert.Equal(t, data.ContentLength, forwardedData.ContentLength, "wrong content length")
	assert.Equal(t, data.Body, forwardedData.Body, "wrong request body")

	// expect all original headers to present in forwarded request (additional headers might be added)
	for k, v := range data.Header {
		fv := forwardedData.Header[k]
		if assert.NotNil(t, fv, "missing expected header: %v = %v", k, v) {
			assert.Equal(t, v, fv, "wrong value of request header: %v", k)
		}
	}
}

func TestRequestData_Forward_ComplexForwardURL(t *testing.T) {
	basket := "zooapi"
	pathSuffix := "/rooms/1/pets/12"

	// Test request
	data := new(RequestData)
	data.Header = make(http.Header)
	data.Header.Add("Content-Type", "application/json")
	data.Header.Add("User-Agent", "Unit-Test")
	data.Method = "PUT"
	data.Body = "{ \"id\" : \"12\", \"kind\" : \"elephant\", \"name\" : \"Bibi\" }"
	data.ContentLength = int64(len(data.Body))
	// path contains basket name
	data.Path = "/" + basket + pathSuffix
	data.Query = "expose=true&pattern=*"

	// Test HTTP server
	var forwardedData *RequestData
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedData = ToRequestData(r)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Config to forward requests to test HTTP server (also enable expanding URL)
	forwardURL := ts.URL + "/captures?from=" + basket
	config := BasketConfig{ForwardURL: forwardURL, ExpandPath: true, Capacity: 20}
	data.Forward(new(http.Client), config, basket)

	// Validate forwarded path
	assert.Equal(t, "/captures"+pathSuffix, forwardedData.Path, "wrong request path")
	assert.Equal(t, "from="+basket+"&"+data.Query, forwardedData.Query, "wrong request query")
}

func TestRequestData_Forward_BrokenURL(t *testing.T) {
	basket := "test"

	// Test request
	data := new(RequestData)
	data.Header = make(http.Header)
	data.Header.Add("Content-Type", "application/json")
	data.Method = "GET"
	data.Body = "{ \"name\" : \"test\", \"action\" : \"add\" }"
	data.ContentLength = int64(len(data.Body))
	// path contains basket name
	data.Path = "/" + basket

	// Config to forward requests to broken URL
	config := BasketConfig{ForwardURL: "abc", ExpandPath: false, Capacity: 20}

	// Should not fail, warning in log is expected
	r, e := data.Forward(new(http.Client), config, basket)
	assert.Nil(t, r, "response is not expected")
	assert.NotNil(t, e, "error is expected")
	assert.Contains(t, e.Error(), "invalid forward URL: abc - parse", "unexpected error message")
	assert.Contains(t, e.Error(), "invalid URI for request", "unexpected error message")
}

func TestRequestData_Forward_UnreachableURL(t *testing.T) {
	basket := "test"

	// Test request
	data := new(RequestData)
	data.Header = make(http.Header)
	data.Header.Add("Content-Type", "application/json")
	data.Method = "GET"
	data.Body = "{ \"name\" : \"test\", \"action\" : \"add\" }"
	data.ContentLength = int64(len(data.Body))
	// path contains basket name
	data.Path = "/" + basket

	// Config to forward requests to unreachable URL
	config := BasketConfig{ForwardURL: "http://localhost:81/should/fail/to/forward", ExpandPath: false, Capacity: 20}

	// Should not fail, warning in log is expected
	r, e := data.Forward(new(http.Client), config, basket)
	assert.Nil(t, e, "error is not expected")
	assert.NotNil(t, r, "response is expected")
	assert.Equal(t, 502, r.StatusCode, "wrong status code")
}

func TestExpandURL(t *testing.T) {
	assert.Equal(t, "/notify/abc/123-123", expandURL("/notify", "/sniffer/abc/123-123", "sniffer"))
	assert.Equal(t, "/hello/world", expandURL("/", "/mybasket/hello/world", "mybasket"))
	assert.Equal(t, "/notify/hello/world", expandURL("/notify", "/notify/hello/world", "notify"))
	assert.Equal(t, "/receive/notification/test/", expandURL("/receive/notification/", "/basket/test/", "basket"))
}

func TestDatabaseStats_Collect(t *testing.T) {
	stats := new(DatabaseStats)
	stats.Collect(&BasketInfo{"a", 5, 10, 100}, 3)
	stats.Collect(&BasketInfo{"b", 5, 30, 200}, 3)
	stats.Collect(&BasketInfo{"c", 5, 5, 300}, 3)
	stats.Collect(&BasketInfo{"d", 0, 0, 400}, 3)
	stats.Collect(&BasketInfo{"e", 5, 20, 500}, 3)
	stats.Collect(&BasketInfo{"f", 10, 40, 600}, 3)
	stats.Collect(&BasketInfo{"g", 0, 0, 700}, 3)
	stats.Collect(&BasketInfo{"h", 5, 5, 800}, 3)

	assert.Equal(t, 8, stats.BasketsCount, "wrong BasketsCount")
	assert.Equal(t, 2, stats.EmptyBasketsCount, "wrong EmptyBasketsCount")
	assert.Equal(t, 40, stats.MaxBasketSize, "wrong MaxBasketSize")
	assert.Equal(t, 35, stats.RequestsCount, "wrong RequestsCount")
	assert.Equal(t, 110, stats.RequestsTotalCount, "wrong RequestsTotalCount")

	assert.Equal(t, 3, len(stats.TopBasketsByDate), "wrong number of TopBasketsByDate")
	assert.Equal(t, "h", stats.TopBasketsByDate[0].Name)
	assert.Equal(t, "g", stats.TopBasketsByDate[1].Name)
	assert.Equal(t, "f", stats.TopBasketsByDate[2].Name)

	assert.Equal(t, 3, len(stats.TopBasketsBySize), "wrong number of TopBasketsBySize")
	assert.Equal(t, "f", stats.TopBasketsBySize[0].Name)
	assert.Equal(t, "b", stats.TopBasketsBySize[1].Name)
	assert.Equal(t, "e", stats.TopBasketsBySize[2].Name)

	// we do not expect avarage basket size, it is not updated automatically
	assert.Equal(t, 0, stats.AvgBasketSize, "unexpected AvgBasketSize")
}

func TestDatabaseStats_UpdateAvarage(t *testing.T) {
	stats := new(DatabaseStats)
	stats.Collect(&BasketInfo{"a", 5, 10, 100}, 3)
	stats.Collect(&BasketInfo{"b", 5, 20, 200}, 3)
	stats.Collect(&BasketInfo{"c", 5, 30, 300}, 3)

	stats.UpdateAvarage()
	assert.Equal(t, 20, stats.AvgBasketSize, "wrong AvgBasketSize")
}

func TestDatabaseStats_UpdateAvarage_Empty(t *testing.T) {
	stats := new(DatabaseStats)
	stats.UpdateAvarage()
	assert.Equal(t, 0, stats.AvgBasketSize, "wrong AvgBasketSize")
}

func test_validateBasketStats(t *testing.T, info *BasketInfo, name string, count int, totalCount int) {
	assert.Equal(t, name, info.Name)
	assert.Equal(t, count, info.RequestsCount, "unexpected requests count for basket: "+name)
	assert.Equal(t, totalCount, info.RequestsTotalCount, "unexpected requests total count for basket: "+name)
	assert.NotEqual(t, int64(0), info.LastRequestDate, "last request date is expected for basket: "+name)
}
