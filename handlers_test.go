package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func testsSetup() {
	// initialize global HTTP clients
	httpClient = new(http.Client)
	httpInsecureClient = new(http.Client)

	// global config
	serverConfig = CreateConfig()
	// global DB (in memory)
	basketsDb = createBasketsDatabase()
}

func testsShutdown() {
	// release global DB
	basketsDb.Release()
}

func TestMain(m *testing.M) {
	testsSetup()
	code := m.Run()
	testsShutdown()
	os.Exit(code)
}

func TestCreateBasket(t *testing.T) {
	basket := "create01"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 201 - created
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		assert.Equal(t, "application/json; charset=UTF-8", w.Header().Get("Content-Type"), "wrong Content-Type")
		assert.Contains(t, w.Body.String(), "\"token\"", "JSON response with token is expected")

		// validate database
		b := basketsDb.Get(basket)
		if assert.NotNil(t, b, "basket '%v' should be created", basket) {
			config := b.Config()
			assert.Equal(t, 200, config.Capacity, "wrong basket capacity")
			assert.False(t, config.InsecureTls, "wrong value of Insecure TLS flag")
			assert.False(t, config.ExpandPath, "wrong value of Expand Path flag")
			assert.Empty(t, config.ForwardUrl, "Forward URL is not expected")
		}
	}
}

func TestCreateBasket_CustomConfig(t *testing.T) {
	basket := "create02"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(
		"{\"capacity\":30,\"insecure_tls\":true,\"expand_path\":true,\"forward_url\": \"http://localhost:12345/test\"}"))

	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 201 - created
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		assert.Equal(t, "application/json; charset=UTF-8", w.Header().Get("Content-Type"), "wrong Content-Type")
		assert.Contains(t, w.Body.String(), "\"token\"", "JSON response with token is expected")

		// validate database
		b := basketsDb.Get(basket)
		if assert.NotNil(t, b, "basket '%v' should be created", basket) {
			config := b.Config()
			assert.Equal(t, 30, config.Capacity, "wrong basket capacity")
			assert.True(t, config.InsecureTls, "wrong value of Insecure TLS flag")
			assert.True(t, config.ExpandPath, "wrong value of Expand Path flag")
			assert.Equal(t, "http://localhost:12345/test", config.ForwardUrl, "wrong Forward URL")
		}
	}
}

func TestCreateBasket_Forbidden(t *testing.T) {
	basket := WEB_ROOT

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 403 - forbidden
		assert.Equal(t, 403, w.Code, "wrong HTTP result code")
		assert.Equal(t, "Basket name may not clash with system path: "+basket+"\n", w.Body.String(), "wrong error message")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestCreateBasket_InvalidName(t *testing.T) {
	basket := ">>>"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 400 - bad request
		assert.Equal(t, 400, w.Code, "wrong HTTP result code")
		assert.Equal(t, "Basket name does not match pattern: "+validBasketName.String()+"\n", w.Body.String(),
			"wrong error message")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestCreateBasket_Conflict(t *testing.T) {
	basket := "create03"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(httptest.NewRecorder(), r, ps)

		// create another basket with the same name
		w := httptest.NewRecorder()
		CreateBasket(w, r, ps)

		// validate response: 409 - conflict
		assert.Equal(t, 409, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "already exists", "error message is incomplete")
	}
}

func TestCreateBasket_InvalidCapacity(t *testing.T) {
	basket := "create04"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket,
		strings.NewReader("{\"capacity\": -10}"))

	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 422 - unprocessable entity
		assert.Equal(t, 422, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "Capacity should be a positive number", "error message is incomplete")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestCreateBasket_ExceedCapacityLimit(t *testing.T) {
	basket := "create05"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket,
		strings.NewReader("{\"capacity\": 10000000}"))

	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 422 - unprocessable entity
		assert.Equal(t, 422, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "Capacity may not be greater than", "error message is incomplete")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestCreateBasket_InvalidForwardUrl(t *testing.T) {
	basket := "create06"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket,
		strings.NewReader("{\"forward_url\": \".,?-7\"}"))

	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 422 - unprocessable entity
		assert.Equal(t, 422, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "invalid URI", "error message is incomplete")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestCreateBasket_BrokenJson(t *testing.T) {
	basket := "create07"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket,
		strings.NewReader("{\"capacity\": 300, "))

	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 400 - bad request
		assert.Equal(t, 400, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "unexpected end of JSON input", "error message is incomplete")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestGetBasket(t *testing.T) {
	basket := "get01"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("GET", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				GetBasket(w, r, ps)

				// validate response: 200 - OK
				assert.Equal(t, 200, w.Code, "wrong HTTP result code")
				assert.Equal(t, "application/json; charset=UTF-8", w.Header().Get("Content-Type"), "wrong Content-Type")

				config := new(BasketConfig)
				err = json.Unmarshal(w.Body.Bytes(), config)
				if assert.NoError(t, err, "Failed to parse GetBasket response") {
					assert.Equal(t, 200, config.Capacity, "wrong basket capacity")
					assert.False(t, config.InsecureTls, "wrong value of Insecure TLS flag")
					assert.False(t, config.ExpandPath, "wrong value of Expand Path flag")
					assert.Empty(t, config.ForwardUrl, "Forward URL is not expected")
				}
			}
		}
	}
}

func TestGetBasket_Unauthorized(t *testing.T) {
	basket := "get02"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		r, err = http.NewRequest("GET", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
		if assert.NoError(t, err) {
			w = httptest.NewRecorder()
			GetBasket(w, r, ps)

			// validate response: 401 - unauthorized
			assert.Equal(t, 401, w.Code, "wrong HTTP result code")
		}
	}
}

func TestGetBasket_WrongToken(t *testing.T) {
	basket := "get03"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		r, err = http.NewRequest("GET", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
		if assert.NoError(t, err) {
			r.Header.Add("Authorization", "wrong_token")
			w = httptest.NewRecorder()
			GetBasket(w, r, ps)

			// validate response: 401 - unauthorized
			assert.Equal(t, 401, w.Code, "wrong HTTP result code")
		}
	}
}

func TestGetBasket_NotFound(t *testing.T) {
	basket := "get04"

	r, err := http.NewRequest("GET", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		r.Header.Add("Authorization", "abcd12345")
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		GetBasket(w, r, ps)

		// validate response: 404 - not found
		assert.Equal(t, 404, w.Code, "wrong HTTP result code")
	}
}

func TestUpdateBasket(t *testing.T) {
	basket := "update01"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("PUT", "http://localhost:55555/baskets/"+basket,
				strings.NewReader("{\"capacity\":50, \"expand_path\":true, \"forward_url\":\"http://test.server/forward\"}"))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				UpdateBasket(w, r, ps)

				// validate response: 204 - no content
				assert.Equal(t, 204, w.Code, "wrong HTTP result code")

				// validate update
				config := basketsDb.Get(basket).Config()
				assert.Equal(t, 50, config.Capacity, "wrong basket capacity")
				assert.False(t, config.InsecureTls, "wrong value of Insecure TLS flag")
				assert.True(t, config.ExpandPath, "wrong value of Expand Path flag")
				assert.Equal(t, "http://test.server/forward", config.ForwardUrl, "wrong Forward URL")
			}
		}
	}
}

func TestUpdateBasket_EmptyConfig(t *testing.T) {
	basket := "update02"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("PUT", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				UpdateBasket(w, r, ps)

				// validate response: 304 - not modified
				assert.Equal(t, 304, w.Code, "wrong HTTP result code")

				// validate update
				config := basketsDb.Get(basket).Config()
				assert.Equal(t, 200, config.Capacity, "wrong basket capacity")
				assert.False(t, config.InsecureTls, "wrong value of Insecure TLS flag")
				assert.False(t, config.ExpandPath, "wrong value of Expand Path flag")
				assert.Empty(t, config.ForwardUrl, "Forward URL is not expected")
			}
		}
	}
}

func TestUpdateBasket_BrokenJson(t *testing.T) {
	basket := "update03"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("PUT", "http://localhost:55555/baskets/"+basket, strings.NewReader("{ capacity : 300 /"))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				UpdateBasket(w, r, ps)

				// validate response: 400 - bad request
				assert.Equal(t, 400, w.Code, "wrong HTTP result code")

				// validate update
				config := basketsDb.Get(basket).Config()
				assert.Equal(t, 200, config.Capacity, "wrong basket capacity")
				assert.False(t, config.InsecureTls, "wrong value of Insecure TLS flag")
				assert.False(t, config.ExpandPath, "wrong value of Expand Path flag")
				assert.Empty(t, config.ForwardUrl, "Forward URL is not expected")
			}
		}
	}
}

func TestDeleteBasket(t *testing.T) {
	basket := "delete01"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		assert.NotNil(t, basketsDb.Get(basket), "basket '%v' is expected", basket)

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("DELETE", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				DeleteBasket(w, r, ps)

				// validate response: 204 - no content
				assert.Equal(t, 204, w.Code, "wrong HTTP result code")

				// validate deletion
				assert.Nil(t, basketsDb.Get(basket), "basket '%v' is not expected", basket)
			}
		}
	}
}

func TestDeleteBasket_NotFound(t *testing.T) {
	basket := "delete02"

	r, err := http.NewRequest("DELETE", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		r.Header.Add("Authorization", "abc123")

		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()
		DeleteBasket(w, r, ps)

		// validate response: 404 - not found
		assert.Equal(t, 404, w.Code, "wrong HTTP result code")
	}
}

func TestDeleteBasket_Unauthorized(t *testing.T) {
	basket := "delete03"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		assert.NotNil(t, basketsDb.Get(basket), "basket '%v' is expected", basket)

		r, err = http.NewRequest("DELETE", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
		if assert.NoError(t, err) {
			r.Header.Add("Authorization", "123-wrong-token")
			w = httptest.NewRecorder()
			DeleteBasket(w, r, ps)

			// validate response: 401 - unauthorized
			assert.Equal(t, 401, w.Code, "wrong HTTP result code")

			// validate not deleted
			assert.NotNil(t, basketsDb.Get(basket), "basket '%v' is expected", basket)
		}
	}
}

func TestGetBaskets(t *testing.T) {
	// create 5 baskets
	for i := 0; i < 5; i++ {
		basket := fmt.Sprintf("names0%v", i)
		r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
		if assert.NoError(t, err) {
			w := httptest.NewRecorder()
			ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
			CreateBasket(w, r, ps)
			assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		}
	}

	// get names
	r, err := http.NewRequest("GET", "http://localhost:55555/baskets", strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		GetBaskets(w, r, make(httprouter.Params, 0))
		// HTTP 200 - OK
		assert.Equal(t, 200, w.Code, "wrong HTTP result code")

		names := new(BasketNamesPage)
		err = json.Unmarshal(w.Body.Bytes(), names)
		if assert.NoError(t, err) {
			// validate response
			assert.NotEmpty(t, names.Names, "names are expected")
			assert.True(t, names.Count > 0, "count should be greater than 0")
		}
	}
}

func TestGetBaskets_Query(t *testing.T) {
	// create 10 baskets
	for i := 0; i < 10; i++ {
		basket := fmt.Sprintf("names1%v", i)
		r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
		if assert.NoError(t, err) {
			w := httptest.NewRecorder()
			ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
			CreateBasket(w, r, ps)
			assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		}
	}

	// get names
	r, err := http.NewRequest("GET", "http://localhost:55555/baskets?q=names1", strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		GetBaskets(w, r, make(httprouter.Params, 0))
		// HTTP 200 - OK
		assert.Equal(t, 200, w.Code, "wrong HTTP result code")

		names := new(BasketNamesQueryPage)
		err = json.Unmarshal(w.Body.Bytes(), names)
		if assert.NoError(t, err) {
			// validate response
			assert.NotEmpty(t, names.Names, "names are expected")
			assert.Len(t, names.Names, 10, "unexpected number of found baskets")
			assert.False(t, names.HasMore, "no more names are expected")
		}
	}
}

func TestGetBaskets_Page(t *testing.T) {
	// create 10 baskets
	for i := 0; i < 10; i++ {
		basket := fmt.Sprintf("names2%v", i)
		r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
		if assert.NoError(t, err) {
			w := httptest.NewRecorder()
			ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
			CreateBasket(w, r, ps)
			assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		}
	}

	// get names
	r, err := http.NewRequest("GET", "http://localhost:55555/baskets?max=5&skip=2", strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		GetBaskets(w, r, make(httprouter.Params, 0))
		// HTTP 200 - OK
		assert.Equal(t, 200, w.Code, "wrong HTTP result code")

		names := new(BasketNamesPage)
		err = json.Unmarshal(w.Body.Bytes(), names)
		if assert.NoError(t, err) {
			// validate response
			assert.NotEmpty(t, names.Names, "names are expected")
			assert.Len(t, names.Names, 5, "unexpected number of found baskets")
			assert.Equal(t, names.Count, basketsDb.Size(), "wrong count of baskets")
			assert.True(t, names.HasMore, "more names are expected")
		}
	}
}

func TestGetBasketRequests(t *testing.T) {
	basket := "getreq01"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		assert.NotNil(t, basketsDb.Get(basket), "basket '%v' is expected", basket)

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			// collect some HTTP requests
			for i := 1; i <= 10; i++ {
				req := createTestPOSTRequest(fmt.Sprintf("http://localhost:55555/%v/data?id=%v", basket, i),
					fmt.Sprintf("req%v data ...", i), "text/plain")
				AcceptBasketRequests(httptest.NewRecorder(), req)
			}

			// get requests
			r, err = http.NewRequest("GET", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				GetBasketRequests(w, r, ps)
				// HTTP 200 - OK
				assert.Equal(t, 200, w.Code, "wrong HTTP result code")

				requests := new(RequestsPage)
				err = json.Unmarshal(w.Body.Bytes(), requests)
				if assert.NoError(t, err) {
					// validate response
					assert.NotEmpty(t, requests.Requests, "requests are expected")
					assert.Len(t, requests.Requests, 10, "unexpected number of returned requests")
					assert.Equal(t, requests.Count, 10, "wrong count of requests")
					assert.Equal(t, requests.TotalCount, 10, "wrong total count of requests")
					assert.False(t, requests.HasMore, "no more requests are expected")
				}
			}
		}
	}
}

func TestGetBasketRequests_Query(t *testing.T) {
	basket := "getreq02"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		assert.NotNil(t, basketsDb.Get(basket), "basket '%v' is expected", basket)

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			// collect some HTTP requests
			for i := 1; i <= 25; i++ {
				req := createTestPOSTRequest(fmt.Sprintf("http://localhost:55555/%v/data?id=%v", basket, i),
					fmt.Sprintf("req%v data ...", i), "text/plain")
				if i > 10 && i < 15 {
					req.Header.Add("Test-Key", "magic")
				}
				AcceptBasketRequests(httptest.NewRecorder(), req)
			}

			// get requests
			r, err = http.NewRequest("GET", "http://localhost:55555/baskets/"+basket+"?q=magic&in=headers", strings.NewReader(""))
			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				GetBasketRequests(w, r, ps)
				// HTTP 200 - OK
				assert.Equal(t, 200, w.Code, "wrong HTTP result code")

				requests := new(RequestsQueryPage)
				err = json.Unmarshal(w.Body.Bytes(), requests)
				if assert.NoError(t, err) {
					// validate response
					assert.NotEmpty(t, requests.Requests, "requests are expected")
					assert.Len(t, requests.Requests, 4, "unexpected number of returned requests")
					assert.False(t, requests.HasMore, "no more requests are expected")
				}
			}
		}
	}
}

func TestGetBasketRequests_Page(t *testing.T) {
	basket := "getreq03"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		assert.NotNil(t, basketsDb.Get(basket), "basket '%v' is expected", basket)

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			// collect some HTTP requests
			for i := 1; i <= 300; i++ {
				req := createTestPOSTRequest(fmt.Sprintf("http://localhost:55555/%v/data?id=%v", basket, i),
					fmt.Sprintf("req%v data ...", i), "text/plain")
				AcceptBasketRequests(httptest.NewRecorder(), req)
			}

			// get requests
			r, err = http.NewRequest("GET", "http://localhost:55555/baskets/"+basket+"?max=5&skip=5", strings.NewReader(""))
			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				GetBasketRequests(w, r, ps)
				// HTTP 200 - OK
				assert.Equal(t, 200, w.Code, "wrong HTTP result code")

				requests := new(RequestsPage)
				err = json.Unmarshal(w.Body.Bytes(), requests)
				if assert.NoError(t, err) {
					// validate response
					assert.NotEmpty(t, requests.Requests, "requests are expected")
					assert.Len(t, requests.Requests, 5, "unexpected number of returned requests")
					assert.Equal(t, requests.Count, 200, "wrong count of requests")
					assert.Equal(t, requests.TotalCount, 300, "wrong total count of requests")
					assert.True(t, requests.HasMore, "more requests are expected")

					assert.Contains(t, requests.Requests[0].Body, "req295", "wrong request")
				}
			}
		}
	}
}

func TestClearBasket(t *testing.T) {
	basket := "clear01"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		assert.NotNil(t, basketsDb.Get(basket), "basket '%v' is expected", basket)

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			// collect some HTTP requests
			for i := 1; i <= 25; i++ {
				req := createTestPOSTRequest(fmt.Sprintf("http://localhost:55555/%v/data?id=%v", basket, i),
					fmt.Sprintf("req%v data ...", i), "text/plain")
				AcceptBasketRequests(httptest.NewRecorder(), req)
			}

			// clear basket
			r, err = http.NewRequest("DELETE", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				ClearBasket(w, r, ps)
				// HTTP 204 - no content
				assert.Equal(t, 204, w.Code, "wrong HTTP result code")

				// get requests
				r, err = http.NewRequest("GET", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
				if assert.NoError(t, err) {
					r.Header.Add("Authorization", auth.Token)
					w = httptest.NewRecorder()
					GetBasketRequests(w, r, ps)
					// HTTP 200 - OK
					assert.Equal(t, 200, w.Code, "wrong HTTP result code")

					requests := new(RequestsPage)
					err = json.Unmarshal(w.Body.Bytes(), requests)
					if assert.NoError(t, err) {
						// validate response
						assert.Empty(t, requests.Requests, "requests are not expected")
						assert.Equal(t, requests.Count, 0, "wrong count of requests")
						assert.Equal(t, requests.TotalCount, 0, "wrong total count of requests")
						assert.False(t, requests.HasMore, "no more requests are expected")
					}
				}
			}
		}
	}
}

func TestAcceptBasketRequests_NotFound(t *testing.T) {
	basket := "accept02"
	req := createTestPOSTRequest("http://localhost:55555/"+basket, "super-data", "text/plain")
	w := httptest.NewRecorder()
	AcceptBasketRequests(w, req)
	// HTTP 404 - not found
	assert.Equal(t, 404, w.Code, "wrong HTTP result code")
}

func TestForwardToWeb(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:55555/", strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ForwardToWeb(w, r, make(httprouter.Params, 0))

		// validate response: 302 - Found
		assert.Equal(t, 302, w.Code, "wrong HTTP result code")
		assert.Equal(t, "/"+WEB_ROOT, w.Header().Get("Location"), "wrong Location header")
	}
}

func TestWebIndexPage(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:55555/web", strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		WebIndexPage(w, r, make(httprouter.Params, 0))

		// validate response: 200 - OK
		assert.Equal(t, 200, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "<title>Request Baskets</title>", "HTML index page with baskets is expected")
	}
}

func TestWebBasketPage(t *testing.T) {
	basket := "test"

	r, err := http.NewRequest("GET", "http://localhost:55555/web/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		WebBasketPage(w, r, ps)

		// validate response: 200 - OK
		assert.Equal(t, 200, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "<title>Request Basket: "+basket+"</title>",
			"HTML page to display basket is expected")
	}
}

func TestWebBasketPage_InvalidName(t *testing.T) {
	basket := ">>>"

	r, err := http.NewRequest("GET", "http://localhost:55555/web/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		WebBasketPage(w, r, ps)

		// validate response: 400 - bad request
		assert.Equal(t, 400, w.Code, "wrong HTTP result code")
		assert.Equal(t, "Basket name does not match pattern: "+validBasketName.String()+"\n", w.Body.String(),
			"wrong error message")
	}
}
