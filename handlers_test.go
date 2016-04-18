package main

import (
	"encoding/json"
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
		assert.Contains(t, w.Body.String(), "Invalid basket name: "+basket, "error message is incomplete")
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

		// validate response: 422 - unprocessable entity
		assert.Equal(t, 422, w.Code, "wrong HTTP result code")
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
