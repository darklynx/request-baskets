package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/boltdb/bolt"
	"github.com/stretchr/testify/assert"
)

func TestToOpts(t *testing.T) {
	assert.Equal(t, []byte{0}, toOpts(BasketConfig{ExpandPath: false, InsecureTLS: false, ProxyResponse: false}), "wrong options")
	assert.Equal(t, []byte{1}, toOpts(BasketConfig{ExpandPath: true, InsecureTLS: false, ProxyResponse: false}), "wrong options")
	assert.Equal(t, []byte{2}, toOpts(BasketConfig{ExpandPath: false, InsecureTLS: true, ProxyResponse: false}), "wrong options")
	assert.Equal(t, []byte{4}, toOpts(BasketConfig{ExpandPath: false, InsecureTLS: false, ProxyResponse: true}), "wrong options")
	assert.Equal(t, []byte{7}, toOpts(BasketConfig{ExpandPath: true, InsecureTLS: true, ProxyResponse: true}), "wrong options")
}

func TestFromOpts(t *testing.T) {
	config := BasketConfig{}
	// default
	fromOpts([]byte{}, &config)
	assert.False(t, config.ExpandPath, "wrong 'ExpandPath' value")
	assert.False(t, config.InsecureTLS, "wrong 'InsecureTLS' value")
	assert.False(t, config.ProxyResponse, "wrong 'ProxyResponse' value")

	// reset
	fromOpts([]byte{0}, &config)
	assert.False(t, config.ExpandPath, "wrong 'ExpandPath' value")
	assert.False(t, config.InsecureTLS, "wrong 'InsecureTLS' value")
	assert.False(t, config.ProxyResponse, "wrong 'ProxyResponse' value")

	// toOpts => fromOpts
	fromOpts(toOpts(BasketConfig{ExpandPath: true, InsecureTLS: false, ProxyResponse: false}), &config)
	assert.True(t, config.ExpandPath, "wrong 'ExpandPath' value")
	assert.False(t, config.InsecureTLS, "wrong 'InsecureTLS' value")
	assert.False(t, config.ProxyResponse, "wrong 'ProxyResponse' value")

	// toOpts => fromOpts
	fromOpts(toOpts(BasketConfig{ExpandPath: false, InsecureTLS: true, ProxyResponse: false}), &config)
	assert.False(t, config.ExpandPath, "wrong 'ExpandPath' value")
	assert.True(t, config.InsecureTLS, "wrong 'InsecureTLS' value")
	assert.False(t, config.ProxyResponse, "wrong 'ProxyResponse' value")

	// toOpts => fromOpts
	fromOpts(toOpts(BasketConfig{ExpandPath: false, InsecureTLS: false, ProxyResponse: true}), &config)
	assert.False(t, config.ExpandPath, "wrong 'ExpandPath' value")
	assert.False(t, config.InsecureTLS, "wrong 'InsecureTLS' value")
	assert.True(t, config.ProxyResponse, "wrong 'ProxyResponse' value")
}

func TestBoltDatabase_Create(t *testing.T) {
	name := "test1"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	auth, err := db.Create(name, BasketConfig{Capacity: 20})
	if assert.NoError(t, err) {
		assert.NotEmpty(t, auth.Token, "basket token may not be empty")
		assert.False(t, len(auth.Token) < 30, "weak basket token: %v", auth.Token)
	}
}

func TestBoltDatabase_Create_NameConflict(t *testing.T) {
	name := "test2"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	db.Create(name, BasketConfig{Capacity: 20})
	auth, err := db.Create(name, BasketConfig{Capacity: 20})

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), ": "+name+" ", "error is not detailed enough")
		assert.Empty(t, auth.Token, "basket token is not expected")
	}
}

func TestBoltDatabase_Get(t *testing.T) {
	name := "test3"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	auth, err := db.Create(name, BasketConfig{Capacity: 16})
	assert.NoError(t, err)

	basket := db.Get(name)
	if assert.NotNil(t, basket, "basket with name: %v is expected", name) {
		assert.True(t, basket.Authorize(auth.Token), "basket authorization has failed")
		assert.Equal(t, 16, basket.Config().Capacity, "wrong capacity")
	}
}

func TestBoltDatabase_Get_NotFound(t *testing.T) {
	name := "test4"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	basket := db.Get(name)
	assert.Nil(t, basket, "basket with name: %v is not expected", name)
}

func TestBoltDatabase_Delete(t *testing.T) {
	name := "test5"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	db.Create(name, BasketConfig{Capacity: 10})
	assert.NotNil(t, db.Get(name), "basket with name: %v is expected", name)

	db.Delete(name)
	assert.Nil(t, db.Get(name), "basket with name: %v is not expected", name)
}

func TestBoltDatabase_Delete_Multi(t *testing.T) {
	name := "test6"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	config := BasketConfig{Capacity: 10}
	for i := 0; i < 10; i++ {
		db.Create(fmt.Sprintf("test%v", i), config)
	}

	assert.NotNil(t, db.Get(name), "basket with name: %v is expected", name)
	assert.Equal(t, 10, db.Size(), "wrong database size")

	db.Delete(name)

	assert.Nil(t, db.Get(name), "basket with name: %v is not expected", name)
	assert.Equal(t, 9, db.Size(), "wrong database size")
}

func TestBoltDatabase_Size(t *testing.T) {
	db := NewBoltDatabase("test7.db")
	defer db.Release()
	defer os.Remove("test7.db")

	config := BasketConfig{Capacity: 15}
	for i := 0; i < 25; i++ {
		db.Create(fmt.Sprintf("test%v", i), config)
	}

	assert.Equal(t, 25, db.Size(), "wrong database size")
}

func TestBoltDatabase_GetNames(t *testing.T) {
	db := NewBoltDatabase("test8.db")
	defer db.Release()
	defer os.Remove("test8.db")

	config := BasketConfig{Capacity: 15}
	for i := 0; i < 45; i++ {
		db.Create(fmt.Sprintf("test%v", i), config)
	}

	// Get and validate page 1 (test0, test1, test10, test11, ... - sorted)
	page1 := db.GetNames(10, 0)
	assert.Equal(t, 45, page1.Count, "wrong baskets count")
	assert.True(t, page1.HasMore, "expected more names")
	assert.Len(t, page1.Names, 10, "wrong page size")
	assert.Equal(t, "test10", page1.Names[2], "wrong basket name at index #2")

	// Get and validate page 5 (test5, test6, test7, test8, test9)
	page5 := db.GetNames(10, 40)
	assert.Equal(t, 45, page5.Count, "wrong baskets count")
	assert.False(t, page5.HasMore, "no more names are expected")
	assert.Len(t, page5.Names, 5, "wrong page size")
	assert.Equal(t, "test5", page5.Names[0], "wrong basket name at index #0")

	// Corner cases
	assert.Empty(t, db.GetNames(0, 0).Names, "names are not expected")
	assert.False(t, db.GetNames(5, 40).HasMore, "no more names are expected")
}

func TestBoltDatabase_FindNames(t *testing.T) {
	db := NewBoltDatabase("test9.db")
	defer db.Release()
	defer os.Remove("test9.db")

	config := BasketConfig{Capacity: 5}
	for i := 0; i < 35; i++ {
		db.Create(fmt.Sprintf("test%v", i), config)
	}

	res1 := db.FindNames("test2", 20, 0)
	assert.False(t, res1.HasMore, "no more names are expected")
	assert.Len(t, res1.Names, 11, "wrong number of found names")
	for _, name := range res1.Names {
		assert.Contains(t, name, "test2", "invalid name among search results")
	}

	res2 := db.FindNames("test1", 5, 0)
	assert.True(t, res2.HasMore, "more names are expected")
	assert.Len(t, res2.Names, 5, "wrong number of found names")

	// Corner cases
	assert.Len(t, db.FindNames("test1", 5, 10).Names, 1, "wrong number of returned names")
	assert.Empty(t, db.FindNames("test2", 5, 20).Names, "names in this page are not expected")
	assert.False(t, db.FindNames("test3", 5, 6).HasMore, "no more names are expected")
	assert.False(t, db.FindNames("abc", 5, 0).HasMore, "no more names are expected")
	assert.Empty(t, db.FindNames("xyz", 5, 0).Names, "names are not expected")
}

func TestBoltBasket_Add(t *testing.T) {
	name := "test101"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	db.Create(name, BasketConfig{Capacity: 20})

	basket := db.Get(name)
	if assert.NotNil(t, basket, "basket with name: %v is expected", name) {
		// add 1st HTTP request
		content := "{ \"user\": \"tester\", \"age\": 24 }"
		data := basket.Add(createTestPOSTRequest(
			fmt.Sprintf("http://localhost/%v/demo?name=abc&ver=12", name), content, "application/json"))

		assert.Equal(t, 1, basket.Size(), "wrong basket size")

		// detailed http.Request to RequestData tests should be covered by test of ToRequestData function
		assert.Equal(t, content, data.Body, "wrong body")
		assert.Equal(t, int64(len(content)), data.ContentLength, "wrong content length")

		// add 2nd HTTP request
		basket.Add(createTestPOSTRequest(fmt.Sprintf("http://localhost/%v/demo", name), "Hellow world", "text/plain"))
		assert.Equal(t, 2, basket.Size(), "wrong basket size")
	}
}

func TestBoltBasket_Add_ExceedLimit(t *testing.T) {
	name := "test102"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	db.Create(name, BasketConfig{Capacity: 10})

	basket := db.Get(name)
	if assert.NotNil(t, basket, "basket with name: %v is expected", name) {
		// fill basket
		for i := 0; i < 35; i++ {
			basket.Add(createTestPOSTRequest(
				fmt.Sprintf("http://localhost/%v/demo", name), fmt.Sprintf("test%v", i), "text/plain"))
		}
		assert.Equal(t, 10, basket.Size(), "wrong basket size")
	}
}

func TestBoltBasket_Clear(t *testing.T) {
	name := "test103"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	db.Create(name, BasketConfig{Capacity: 20})

	basket := db.Get(name)
	if assert.NotNil(t, basket, "basket with name: %v is expected", name) {
		// fill basket
		for i := 0; i < 15; i++ {
			basket.Add(createTestPOSTRequest(
				fmt.Sprintf("http://localhost/%v/demo", name), fmt.Sprintf("test%v", i), "text/plain"))
		}
		assert.Equal(t, 15, basket.Size(), "wrong basket size")

		// clean basket
		basket.Clear()
		assert.Equal(t, 0, basket.Size(), "wrong basket size, empty basket is expected")
	}
}

func TestBoltBasket_Update_Shrink(t *testing.T) {
	name := "test104"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	db.Create(name, BasketConfig{Capacity: 30})

	basket := db.Get(name)
	if assert.NotNil(t, basket, "basket with name: %v is expected", name) {
		// fill basket
		for i := 0; i < 25; i++ {
			basket.Add(createTestPOSTRequest(
				fmt.Sprintf("http://localhost/%v/demo", name), fmt.Sprintf("test%v", i), "text/plain"))
		}
		assert.Equal(t, 25, basket.Size(), "wrong basket size")

		// update config with lower capacity
		config := basket.Config()
		config.Capacity = 12
		basket.Update(config)
		assert.Equal(t, config.Capacity, basket.Size(), "wrong basket size")
	}
}

func TestBoltBasket_GetRequests(t *testing.T) {
	name := "test105"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	db.Create(name, BasketConfig{Capacity: 25})

	basket := db.Get(name)
	if assert.NotNil(t, basket, "basket with name: %v is expected", name) {
		// fill basket
		for i := 1; i <= 35; i++ {
			basket.Add(createTestPOSTRequest(
				fmt.Sprintf("http://localhost/%v/demo?id=%v", name, i), fmt.Sprintf("req%v", i), "text/plain"))
		}
		assert.Equal(t, 25, basket.Size(), "wrong basket size")

		// Get and validate last 10 requests
		page1 := basket.GetRequests(10, 0)
		assert.True(t, page1.HasMore, "expected more requests")
		assert.Len(t, page1.Requests, 10, "wrong page size")
		assert.Equal(t, 25, page1.Count, "wrong requests count")
		assert.Equal(t, 35, page1.TotalCount, "wrong requests total count")
		assert.Equal(t, "req35", page1.Requests[0].Body, "last request #35 is expected at index #0")

		// Get and validate 10 requests, skip 20
		page3 := basket.GetRequests(10, 20)
		assert.False(t, page3.HasMore, "no more requests are expected")
		assert.Len(t, page3.Requests, 5, "wrong page size")
		assert.Equal(t, 25, page3.Count, "wrong requests count")
		assert.Equal(t, 35, page3.TotalCount, "wrong requests total count")
		assert.Equal(t, "req15", page3.Requests[0].Body, "request #15 is expected at index #0")
	}
}

func TestBoltBasket_FindRequests(t *testing.T) {
	name := "test106"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	db.Create(name, BasketConfig{Capacity: 100})

	basket := db.Get(name)
	if assert.NotNil(t, basket, "basket with name: %v is expected", name) {
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
		assert.Equal(t, 30, basket.Size(), "wrong basket size")

		// search everywhere
		s1 := basket.FindRequests("req1", "any", 30, 0)
		assert.False(t, s1.HasMore, "no more results are expected")
		assert.Len(t, s1.Requests, 11, "wrong number of found requests")
		for _, r := range s1.Requests {
			assert.Contains(t, r.Body, "req1", "incorrect request among results")
		}

		// search everywhere (limited output)
		s2 := basket.FindRequests("req2", "any", 5, 5)
		assert.True(t, s2.HasMore, "more results are expected")
		assert.Len(t, s2.Requests, 5, "wrong number of found requests")

		// search in body (positive)
		assert.Len(t, basket.FindRequests("req3", "body", 100, 0).Requests, 2, "wrong number of found requests")
		// search in body (negative)
		assert.Empty(t, basket.FindRequests("yummy", "body", 100, 0).Requests, "found unexpected requests")

		// search in headers (positive)
		assert.Len(t, basket.FindRequests("yummy", "headers", 100, 0).Requests, 10, "wrong number of found requests")
		assert.Len(t, basket.FindRequests("tasty", "headers", 100, 0).Requests, 20, "wrong number of found requests")
		// search in headers (negative)
		assert.Empty(t, basket.FindRequests("req1", "headers", 100, 0).Requests, "found unexpected requests")

		// search in query (positive)
		assert.Len(t, basket.FindRequests("id=1", "query", 100, 0).Requests, 11, "wrong number of found requests")
		// search in query (negative)
		assert.Empty(t, basket.FindRequests("tasty", "query", 100, 0).Requests, "found unexpected requests")
	}
}

func TestBoltBasket_SetResponse(t *testing.T) {
	name := "test107"
	method := "POST"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	db.Create(name, BasketConfig{Capacity: 20})

	basket := db.Get(name)
	if assert.NotNil(t, basket, "basket with name: %v is expected", name) {
		// Ensure no response
		assert.Nil(t, basket.GetResponse(method))

		// Set response
		basket.SetResponse(method, ResponseConfig{Status: 201, Body: "{ 'message' : 'created' }"})
		// Get and validate
		response := basket.GetResponse(method)
		if assert.NotNil(t, response, "response for method: %v is expected", method) {
			assert.Equal(t, 201, response.Status, "wrong HTTP response status")
			assert.Equal(t, "{ 'message' : 'created' }", response.Body, "wrong HTTP response body")
			assert.False(t, response.IsTemplate, "template is not expected")
		}
	}
}

func TestBoltBasket_SetResponse_Update(t *testing.T) {
	name := "test108"
	method := "GET"
	db := NewBoltDatabase(name + ".db")
	defer db.Release()
	defer os.Remove(name + ".db")

	db.Create(name, BasketConfig{Capacity: 20})

	basket := db.Get(name)
	if assert.NotNil(t, basket, "basket with name: %v is expected", name) {
		// Set response
		basket.SetResponse(method, ResponseConfig{Status: 200, Body: ""})
		// Update response
		basket.SetResponse(method, ResponseConfig{Status: 200, Body: "welcome", IsTemplate: true})
		// Get and validate
		response := basket.GetResponse(method)
		if assert.NotNil(t, response, "response for method: %v is expected", method) {
			assert.Equal(t, 200, response.Status, "wrong HTTP response status")
			assert.Equal(t, "welcome", response.Body, "wrong HTTP response body")
			assert.True(t, response.IsTemplate, "template is expected")
		}
	}
}

func TestBoltBasket_InvalidBasket(t *testing.T) {
	name := "test199"
	db, _ := bolt.Open(name+".db", 0600, &bolt.Options{Timeout: 5 * time.Second})
	defer db.Close()
	defer os.Remove(name + ".db")

	// create a basket referring non-existing name in the database file
	basket := &boltBasket{db, name}
	// should print error in log file during update
	basket.Clear()
	// should print error in log file during view and return nil
	assert.Nil(t, basket.GetResponse("GET"), "expected to fail and return nil")
}

func TestNewBoltDatabase_Error(t *testing.T) {
	file := "test200.db"
	db := NewBoltDatabase(file)
	if assert.NotNil(t, db, "Bolt database is expected with file name: %s", file) {
		defer db.Release()
		defer os.Remove(file)
		// second attempt to create database with file that already opened should fail
		assert.Nil(t, NewBoltDatabase(file), "expected to fail and return nil")
	}
}
