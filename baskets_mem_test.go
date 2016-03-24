package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func createTestDatabase() BasketsDatabase {
	return NewMemoryDatabase()
}

func createTestPOSTRequest(reqUrl string, content string, contentType string) *http.Request {
	request := new(http.Request)
	request.Method = "POST"
	request.URL, _ = url.Parse(reqUrl)
	request.Body = ioutil.NopCloser(strings.NewReader(content))
	request.ContentLength = int64(len(content))
	request.Header = make(http.Header)
	request.Header.Add("Content-Type", contentType)
	request.Header.Add("User-Agent", "Unit-Test")
	request.Header.Add("Accept", "application/json")

	return request
}

func TestMemoryDatabase_Create(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	name := "test1"
	auth, err := db.Create(name, BasketConfig{Capacity: 20})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(auth.Token) == 0 {
		t.Fatalf("basket token is expected")
	}
	if len(auth.Token) < 30 {
		t.Fatalf("insecure token is generated: %v", auth.Token)
	}
}

func TestMemoryDatabase_Create_NameConflict(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	name := "test2"
	db.Create(name, BasketConfig{Capacity: 20})
	auth, err := db.Create(name, BasketConfig{Capacity: 20})
	if err == nil {
		t.Fatalf("error is expected")
	}
	if !strings.Contains(err.Error(), "'"+name+"'") {
		t.Fatalf("error is not detailed enough: %v", err)
	}
	if len(auth.Token) > 0 {
		t.Fatalf("token is not expected, but was: %v", auth.Token)
	}
}

func TestMemoryDatabase_Get(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	name := "test3"
	auth, err := db.Create(name, BasketConfig{Capacity: 16})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	basket := db.Get(name)
	if basket == nil {
		t.Fatalf("basket with name: %v is expected", name)
	}

	if !basket.Authorize(auth.Token) {
		t.Fatalf("wrong basket, authorization with token: %v has failed", auth.Token)
	}

	if basket.Config().Capacity != 16 {
		t.Fatalf("wrong basket capacity: %v", basket.Config().Capacity)
	}
}

func TestMemoryDatabase_Get_NotFound(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	name := "test4"
	basket := db.Get(name)
	if basket != nil {
		t.Fatalf("basket with name: %v is not expected", name)
	}
}

func TestMemoryDatabase_Delete(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	name := "test5"
	db.Create(name, BasketConfig{Capacity: 10})
	if db.Get(name) == nil {
		t.Fatalf("basket with name: %v is expected", name)
	}

	db.Delete(name)

	if db.Get(name) != nil {
		t.Fatalf("basket with name: %v is not expected", name)
	}
}

func TestMemoryDatabase_Delete_Multi(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	name := "test6"
	config := BasketConfig{Capacity: 10}
	for i := 0; i < 10; i++ {
		db.Create(fmt.Sprintf("test%v", i), config)
	}

	if db.Get(name) == nil {
		t.Fatalf("basket with name: %v is expected", name)
	}
	if db.Size() != 10 {
		t.Fatalf("wrong database size: %v", db.Size())
	}

	db.Delete(name)

	if db.Get(name) != nil {
		t.Fatalf("basket with name: %v is not expected", name)
	}
	if db.Size() != 9 {
		t.Fatalf("wrong database size: %v", db.Size())
	}
}

func TestMemoryDatabase_Size(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	config := BasketConfig{Capacity: 15}
	for i := 0; i < 25; i++ {
		db.Create(fmt.Sprintf("test%v", i), config)
	}

	if db.Size() != 25 {
		t.Fatalf("wrong database size: %v", db.Size())
	}
}

func TestMemoryDatabase_GetNames(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	config := BasketConfig{Capacity: 15}
	for i := 0; i < 45; i++ {
		db.Create(fmt.Sprintf("test%v", i), config)
	}

	// Get and validate page 1 (test0 - test9)
	page1 := db.GetNames(10, 0)
	if page1.Count != 45 {
		t.Fatalf("wrong baskets count: %v", page1.Count)
	}
	if !page1.HasMore {
		t.Fatalf("expected more names")
	}
	if len(page1.Names) != 10 {
		t.Fatalf("wrong page size: %v", len(page1.Names))
	}
	if page1.Names[2] != "test2" {
		t.Fatalf("wrong basket name: %v", page1.Names[2])
	}

	// Get and validate page 5 (test40 - test44)
	page5 := db.GetNames(10, 40)
	if page5.Count != 45 {
		t.Fatalf("wrong baskets count: %v", page5.Count)
	}
	if page5.HasMore {
		t.Fatalf("no more names are expected")
	}
	if len(page5.Names) != 5 {
		t.Fatalf("wrong page size: %v", len(page5.Names))
	}
	if page5.Names[0] != "test40" {
		t.Fatalf("wrong basket name: %v", page5.Names[0])
	}

	// Corner cases
	if len(db.GetNames(0, 0).Names) > 0 {
		t.Fatalf("names are not expected")
	}
	if db.GetNames(5, 40).HasMore {
		t.Fatalf("no more names are expected")
	}
}

func TestMemoryDatabase_FindNames(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	config := BasketConfig{Capacity: 5}
	for i := 0; i < 35; i++ {
		db.Create(fmt.Sprintf("test%v", i), config)
	}

	res1 := db.FindNames("test2", 20, 0)
	if res1.HasMore {
		t.Fatalf("no more names are expected")
	}
	if len(res1.Names) != 11 {
		t.Fatalf("wrong number of found names: %v", len(res1.Names))
	}
	for _, name := range res1.Names {
		if !strings.Contains(name, "test2") {
			t.Fatalf("invalid name among search results: %v", name)
		}
	}

	res2 := db.FindNames("test1", 5, 0)
	if !res2.HasMore {
		t.Fatalf("more names are expected")
	}
	if len(res2.Names) != 5 {
		t.Fatalf("wrong number of returned names: %v", len(res2.Names))
	}

	// Corner cases
	if len(db.FindNames("test1", 5, 10).Names) != 1 {
		t.Fatalf("wrong number of returned names")
	}
	if len(db.FindNames("test2", 5, 20).Names) != 0 {
		t.Fatalf("wrong number of returned names")
	}
	if db.FindNames("test3", 5, 6).HasMore {
		t.Fatalf("no more names are expected")
	}
	if db.FindNames("abc", 5, 0).HasMore {
		t.Fatalf("no more names are expected")
	}
	if len(db.FindNames("xyz", 5, 0).Names) > 0 {
		t.Fatalf("names are not expected")
	}
}

func TestMemoryBasket_Add(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	name := "test101"
	db.Create(name, BasketConfig{Capacity: 20})

	basket := db.Get(name)
	if basket == nil {
		t.Fatalf("basket with name: %v is expected", name)
	}

	// add 1st HTTP request
	content := "{ \"user\": \"tester\", \"age\": 24 }"
	data := basket.Add(createTestPOSTRequest(
		fmt.Sprintf("http://localhost/%v/demo?name=abc&ver=12", name), content, "application/json"))

	if basket.Size() != 1 {
		t.Fatalf("incorrect basket size: %v, expected: 1", basket.Size())
	}

	// detailed http.Request to RequestData tests should be covered by test of ToRequestData function
	if data.Body != content {
		t.Fatalf("unexpected body: %v", data.Body)
	}
	if data.ContentLength != int64(len(content)) {
		t.Fatalf("unexpected content lenght: %v", data.ContentLength)
	}

	// add 2nd HTTP request
	basket.Add(createTestPOSTRequest(fmt.Sprintf("http://localhost/%v/demo", name), "Hellow world", "text/plain"))
	if basket.Size() != 2 {
		t.Fatalf("wrong basket size: %v, expected: 2", basket.Size())
	}
}

func TestMemoryBasket_Add_ExceedLimit(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	name := "test102"
	db.Create(name, BasketConfig{Capacity: 10})

	basket := db.Get(name)
	if basket == nil {
		t.Fatalf("basket with name: %v is expected", name)
	}

	// fill basket
	for i := 0; i < 35; i++ {
		basket.Add(createTestPOSTRequest(
			fmt.Sprintf("http://localhost/%v/demo", name), fmt.Sprintf("test%v", i), "text/plain"))
	}
	if basket.Size() != 10 {
		t.Fatalf("wrong basket size: %v, expected: 10", basket.Size())
	}
}

func TestMemoryBasket_Clear(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	name := "test103"
	db.Create(name, BasketConfig{Capacity: 20})

	basket := db.Get(name)
	if basket == nil {
		t.Fatalf("basket with name: %v is expected", name)
	}

	// fill basket
	for i := 0; i < 15; i++ {
		basket.Add(createTestPOSTRequest(
			fmt.Sprintf("http://localhost/%v/demo", name), fmt.Sprintf("test%v", i), "text/plain"))
	}
	if basket.Size() != 15 {
		t.Fatalf("wrong basket size: %v, expected: 15", basket.Size())
	}

	// clean basket
	basket.Clear()
	if basket.Size() != 0 {
		t.Fatalf("wrong basket size: %v, expected empty basket", basket.Size())
	}
}

func TestMemoryBasket_Update_Shrink(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	name := "test104"
	db.Create(name, BasketConfig{Capacity: 30})

	basket := db.Get(name)
	if basket == nil {
		t.Fatalf("basket with name: %v is expected", name)
	}

	// fill basket
	for i := 0; i < 25; i++ {
		basket.Add(createTestPOSTRequest(
			fmt.Sprintf("http://localhost/%v/demo", name), fmt.Sprintf("test%v", i), "text/plain"))
	}
	if basket.Size() != 25 {
		t.Fatalf("wrong basket size: %v, expected: 25", basket.Size())
	}

	// update config with lower capacity
	config := basket.Config()
	config.Capacity = 12
	basket.Update(config)

	if basket.Size() != 12 {
		t.Fatalf("wrong basket size: %v, expected: 12", basket.Size())
	}
}

func TestMemoryBasket_GetRequests(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	name := "test105"
	db.Create(name, BasketConfig{Capacity: 25})

	basket := db.Get(name)
	if basket == nil {
		t.Fatalf("basket with name: %v is expected", name)
	}

	// fill basket
	for i := 1; i <= 35; i++ {
		basket.Add(createTestPOSTRequest(
			fmt.Sprintf("http://localhost/%v/demo?id=%v", name, i), fmt.Sprintf("req%v", i), "text/plain"))
	}
	if basket.Size() != 25 {
		t.Fatalf("wrong basket size: %v, expected: 25", basket.Size())
	}

	// Get and validate last 10 requests
	page1 := basket.GetRequests(10, 0)
	if !page1.HasMore {
		t.Fatalf("expected more requests")
	}
	if len(page1.Requests) != 10 {
		t.Fatalf("wrong page size: %v", len(page1.Requests))
	}
	if page1.Count != 25 {
		t.Fatalf("wrong requests count: %v", page1.Count)
	}
	if page1.TotalCount != 35 {
		t.Fatalf("wrong total count: %v", page1.Count)
	}
	if page1.Requests[0].Body != "req35" {
		t.Fatalf("last request is expected, but was: %v", page1.Requests[0].Body)
	}

	// Get and validate 10 requests, skip 20
	page3 := basket.GetRequests(10, 20)
	if page3.HasMore {
		t.Fatalf("no more requests are expected")
	}
	if len(page3.Requests) != 5 {
		t.Fatalf("wrong page size: %v", len(page3.Requests))
	}
	if page3.Count != 25 {
		t.Fatalf("wrong requests count: %v", page3.Count)
	}
	if page3.TotalCount != 35 {
		t.Fatalf("wrong total count: %v", page3.Count)
	}
	if page3.Requests[0].Body != "req15" {
		t.Fatalf("15th request is expected, but was: %v", page3.Requests[0].Body)
	}
}

func TestMemoryBasket_FindRequests(t *testing.T) {
	db := createTestDatabase()
	defer db.Release()

	name := "test106"
	db.Create(name, BasketConfig{Capacity: 100})

	basket := db.Get(name)
	if basket == nil {
		t.Fatalf("basket with name: %v is expected", name)
	}

	// fill basket
	for i := 1; i <= 30; i++ {
		r := createTestPOSTRequest(fmt.Sprintf("http://localhost/%v?id=%v", name, i), fmt.Sprintf("req%v", i), "text/plain")
		r.Header.Add("HeaderId", fmt.Sprintf("header%v", i))
		if i <= 10 {
			r.Header.Add("ChocoPie", "yummy")
		}
		if i <= 20 {
			r.Header.Add("Muffin", "tasty")
		}
		basket.Add(r)
	}
	if basket.Size() != 30 {
		t.Fatalf("wrong basket size: %v, expected: 30", basket.Size())
	}

	// search everywhere
	s1 := basket.FindRequests("req1", "any", 30, 0)
	if s1.HasMore {
		t.Fatalf("no more results are expected")
	}
	if len(s1.Requests) != 11 {
		t.Fatalf("wrong number of found requests: %v", len(s1.Requests))
	}
	for _, r := range s1.Requests {
		if !strings.Contains(r.Body, "req1") {
			t.Fatalf("incorrect request: %v", r.Body)
		}
	}

	// search everywhere (limited output)
	s2 := basket.FindRequests("req2", "any", 5, 5)
	if !s2.HasMore {
		t.Fatalf("more results are expected")
	}
	if len(s2.Requests) != 5 {
		t.Fatalf("wrong number of found requests: %v", len(s2.Requests))
	}

	// search in body (positive)
	if len(basket.FindRequests("req3", "body", 100, 0).Requests) != 2 {
		t.Fatalf("expected requests are not found")
	}
	// search in body (negative)
	if len(basket.FindRequests("yummy", "body", 100, 0).Requests) != 0 {
		t.Fatalf("found unexpected requests")
	}

	// search in headers (positive)
	if len(basket.FindRequests("yummy", "headers", 100, 0).Requests) != 10 {
		t.Fatalf("expected requests are not found")
	}
	if len(basket.FindRequests("tasty", "headers", 100, 0).Requests) != 20 {
		t.Fatalf("expected requests are not found")
	}
	// search in headers (negative)
	if len(basket.FindRequests("req1", "headers", 100, 0).Requests) != 0 {
		t.Fatalf("found unexpected requests")
	}

	// search in query (positive)
	if len(basket.FindRequests("id=1", "query", 100, 0).Requests) != 11 {
		t.Fatalf("expected requests are not found")
	}
	// search in query (negative)
	if len(basket.FindRequests("tasty", "query", 100, 0).Requests) != 0 {
		t.Fatalf("found unexpected requests")
	}
}
