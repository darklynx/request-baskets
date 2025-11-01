package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"io/ioutil"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusCreated, []byte("{\"token\":\"qwerty12345\"}"), nil)
	assert.Equal(t, 201, w.Code, "wrong HTTP response code")
	assert.Equal(t, "{\"token\":\"qwerty12345\"}", w.Body.String(), "wrong HTTP response body")
	assert.Equal(t, "application/json; charset=UTF-8", w.Header().Get("Content-Type"), "wrong Content-Type")
}

func TestWriteJSON_Error(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, nil, fmt.Errorf("Failed to generate JSON: whatever reason"))
	assert.Equal(t, 500, w.Code, "wrong HTTP response code")
	assert.Equal(t, "Failed to generate JSON: whatever reason\n", w.Body.String(), "wrong HTTP response body")
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"), "wrong Content-Type")
}

func TestParseInt(t *testing.T) {
	assert.Equal(t, 12, parseInt("12", 1, 100, 50))
	assert.Equal(t, 50, parseInt("abc", 1, 100, 50))
	// out of range
	assert.Equal(t, 1, parseInt("0", 1, 100, 50))
	assert.Equal(t, 1, parseInt("-10", 1, 100, 50))
	assert.Equal(t, 100, parseInt("500", 1, 100, 50))
}

func TestCreateBasket(t *testing.T) {
	basket := "create01"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
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
			assert.False(t, config.InsecureTLS, "wrong value of Insecure TLS flag")
			assert.False(t, config.ExpandPath, "wrong value of Expand Path flag")
			assert.Empty(t, config.ForwardURL, "Forward URL is not expected")
		}
	}
}

func TestCreateBasket_CustomConfig(t *testing.T) {
	basket := "create02"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(
		"{\"capacity\":30,\"insecure_tls\":true,\"expand_path\":true,\"forward_url\": \"http://localhost:12345/test\",\"proxy_response\":true}"))

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
			assert.True(t, config.InsecureTLS, "wrong value of Insecure TLS flag")
			assert.True(t, config.ExpandPath, "wrong value of Expand Path flag")
			assert.True(t, config.ProxyResponse, "wrong value of Proxy Response flag")
			assert.Equal(t, "http://localhost:12345/test", config.ForwardURL, "wrong Forward URL")
		}
	}
}

func TestCreateBasket_Forbidden(t *testing.T) {
	basket := serviceUIPath

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 403 - forbidden
		assert.Equal(t, 403, w.Code, "wrong HTTP result code")
		assert.Equal(t, "This basket name conflicts with reserved system path: "+basket+"\n", w.Body.String(), "wrong error message")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestCreateBasket_InvalidName(t *testing.T) {
	basket := ">>>"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 400 - Bad Request
		assert.Equal(t, 400, w.Code, "wrong HTTP result code")
		assert.Equal(t, "invalid basket name; the name does not match pattern: "+validBasketName.String()+"\n", w.Body.String(),
			"wrong error message")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestCreateBasket_Conflict(t *testing.T) {
	basket := "create03"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(httptest.NewRecorder(), r, ps)

		// create another basket with the same name
		w := httptest.NewRecorder()
		CreateBasket(w, r, ps)

		// validate response: 409 - Conflict
		assert.Equal(t, 409, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "already exists", "error message is incomplete")
	}
}

func TestCreateBasket_InvalidCapacity(t *testing.T) {
	basket := "create04"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket,
		strings.NewReader("{\"capacity\": -10}"))

	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 422 - Unprocessable Entity
		assert.Equal(t, 422, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "capacity should be a positive number", "error message is incomplete")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestCreateBasket_ExceedCapacityLimit(t *testing.T) {
	basket := "create05"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket,
		strings.NewReader("{\"capacity\": 10000000}"))

	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 422 - Unprocessable Entity
		assert.Equal(t, 422, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "capacity may not be greater than", "error message is incomplete")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestCreateBasket_InvalidForwardUrl(t *testing.T) {
	basket := "create06"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket,
		strings.NewReader("{\"forward_url\": \".,?-7\"}"))

	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 422 - Unprocessable Entity
		assert.Equal(t, 422, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "invalid URI", "error message is incomplete")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestCreateBasket_BrokenJson(t *testing.T) {
	basket := "create07"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket,
		strings.NewReader("{\"capacity\": 300, "))

	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 400 - Bad Request
		assert.Equal(t, 400, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "unexpected end of JSON input", "error message is incomplete")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestCreateBasket_ConfigOutOfLimit(t *testing.T) {
	basket := "create08"

	// only first 2048 bytes of config are read, bigger amount is truncated; this leads to an invalid JSON
	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket,
		strings.NewReader("{\"capacity\": 300, \"forward_url\": \"http://localhost:8080/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/"+
			"1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890/1234567890abcd\"}"))

	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 400 - Bad Request
		assert.Equal(t, 400, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "unexpected end of JSON input", "error message is incomplete")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestCreateBasket_ReadTimeout(t *testing.T) {
	basket := "create09"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket,
		iotest.TimeoutReader(strings.NewReader("{\"capacity\": 300, \"forward_url\": \"http://localhost:8080/1234567890\"}")))

	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 500 - Internal Server Error
		assert.Equal(t, 500, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "timeout\n", "error message is incomplete")
		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}
}

func TestCreateBasket_Unauthorized(t *testing.T) {
	basket := "create10"

	serverConfig.Mode = ModeRestricted
	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 401 - Unauthorized
		assert.Equal(t, 401, w.Code, "wrong HTTP result code")

		// validate database
		assert.Nil(t, basketsDb.Get(basket), "basket '%v' should not be created", basket)
	}

	serverConfig.Mode = ModePublic
}

func TestCreateBasket_Authorized(t *testing.T) {
	basket := "create11"

	serverConfig.Mode = ModeRestricted
	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		r.Header.Add("Authorization", serverConfig.MasterToken)

		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response: 201 - Created
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// validate database
		assert.NotNil(t, basketsDb.Get(basket), "basket '%v' should be created", basket)
	}

	serverConfig.Mode = ModePublic
}

func TestGetBasket(t *testing.T) {
	basket := "get01"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("GET", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
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
					assert.False(t, config.InsecureTLS, "wrong value of Insecure TLS flag")
					assert.False(t, config.ExpandPath, "wrong value of Expand Path flag")
					assert.Empty(t, config.ForwardURL, "Forward URL is not expected")
				}
			}
		}
	}
}

func TestGetBasket_Unauthorized(t *testing.T) {
	basket := "get02"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		r, err = http.NewRequest("GET", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
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

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		r, err = http.NewRequest("GET", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
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

	r, err := http.NewRequest("GET", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		r.Header.Add("Authorization", "abcd12345")
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		GetBasket(w, r, ps)

		// validate response: 404 - not found
		assert.Equal(t, 404, w.Code, "wrong HTTP result code")
	}
}

func TestGetBasket_BadRequest(t *testing.T) {
	basket := "get05~"

	r, err := http.NewRequest("GET", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		r.Header.Add("Authorization", "abcd12345")
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		GetBasket(w, r, ps)

		// validate response: 400 - Bad Request
		assert.Equal(t, 400, w.Code, "wrong HTTP result code")
		assert.Equal(t, "invalid basket name; the name does not match pattern: "+validBasketName.String()+"\n", w.Body.String(),
			"wrong error message")
	}
}

func TestUpdateBasket(t *testing.T) {
	basket := "update01"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("PUT", "http://localhost:55555/api/baskets/"+basket,
				strings.NewReader("{\"capacity\":50, \"expand_path\":true, \"forward_url\":\"http://test.server/forward\",\"proxy_response\":true}"))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				UpdateBasket(w, r, ps)

				// validate response: 204 - no content
				assert.Equal(t, 204, w.Code, "wrong HTTP result code")

				// validate update
				config := basketsDb.Get(basket).Config()
				assert.Equal(t, 50, config.Capacity, "wrong basket capacity")
				assert.False(t, config.InsecureTLS, "wrong value of Insecure TLS flag")
				assert.True(t, config.ExpandPath, "wrong value of Expand Path flag")
				assert.True(t, config.ProxyResponse, "wrong value of Proxy Response flag")
				assert.Equal(t, "http://test.server/forward", config.ForwardURL, "wrong Forward URL")
			}
		}
	}
}

func TestUpdateBasket_EmptyConfig(t *testing.T) {
	basket := "update02"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("PUT", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				UpdateBasket(w, r, ps)

				// validate response: 304 - Not Modified
				assert.Equal(t, 304, w.Code, "wrong HTTP result code")

				// validate update
				config := basketsDb.Get(basket).Config()
				assert.Equal(t, 200, config.Capacity, "wrong basket capacity")
				assert.False(t, config.InsecureTLS, "wrong value of Insecure TLS flag")
				assert.False(t, config.ExpandPath, "wrong value of Expand Path flag")
				assert.Empty(t, config.ForwardURL, "Forward URL is not expected")
			}
		}
	}
}

func TestUpdateBasket_BrokenJson(t *testing.T) {
	basket := "update03"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("PUT", "http://localhost:55555/api/baskets/"+basket, strings.NewReader("{ capacity : 300 /"))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				UpdateBasket(w, r, ps)

				// validate response: 400 - Bad Request
				assert.Equal(t, 400, w.Code, "wrong HTTP result code")

				// validate update
				config := basketsDb.Get(basket).Config()
				assert.Equal(t, 200, config.Capacity, "wrong basket capacity")
				assert.False(t, config.InsecureTLS, "wrong value of Insecure TLS flag")
				assert.False(t, config.ExpandPath, "wrong value of Expand Path flag")
				assert.Empty(t, config.ForwardURL, "Forward URL is not expected")
			}
		}
	}
}

func TestUpdateBasket_ReadTimeout(t *testing.T) {
	basket := "update04"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("PUT", "http://localhost:55555/api/baskets/"+basket,
				iotest.TimeoutReader(strings.NewReader("{\"capacity\":300}")))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				UpdateBasket(w, r, ps)

				// validate response: 500 - Internal Server Error
				assert.Equal(t, 500, w.Code, "wrong HTTP result code")

				// validate update
				config := basketsDb.Get(basket).Config()
				assert.Equal(t, 200, config.Capacity, "wrong basket capacity")
				assert.False(t, config.InsecureTLS, "wrong value of Insecure TLS flag")
				assert.False(t, config.ExpandPath, "wrong value of Expand Path flag")
				assert.Empty(t, config.ForwardURL, "Forward URL is not expected")
			}
		}
	}
}

func TestUpdateBasket_InvalidConfig(t *testing.T) {
	basket := "update05"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("PUT", "http://localhost:55555/api/baskets/"+basket,
				strings.NewReader("{\"capacity\":50000000}"))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				UpdateBasket(w, r, ps)

				// validate response: 422 - Unprocessable Entity
				assert.Equal(t, 422, w.Code, "wrong HTTP result code")

				// validate update
				config := basketsDb.Get(basket).Config()
				assert.Equal(t, 200, config.Capacity, "wrong basket capacity")
				assert.False(t, config.InsecureTLS, "wrong value of Insecure TLS flag")
				assert.False(t, config.ExpandPath, "wrong value of Expand Path flag")
				assert.Empty(t, config.ForwardURL, "Forward URL is not expected")
			}
		}
	}
}

func TestDeleteBasket(t *testing.T) {
	basket := "delete01"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
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
			r, err = http.NewRequest("DELETE", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))

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

	r, err := http.NewRequest("DELETE", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
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

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		assert.NotNil(t, basketsDb.Get(basket), "basket '%v' is expected", basket)

		r, err = http.NewRequest("DELETE", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
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
		r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
		if assert.NoError(t, err) {
			w := httptest.NewRecorder()
			ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
			CreateBasket(w, r, ps)
			assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		}
	}

	// get names
	r, err := http.NewRequest("GET", "http://localhost:55555/api/baskets", strings.NewReader(""))
	if assert.NoError(t, err) {
		r.Header.Add("Authorization", serverConfig.MasterToken)
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

func TestGetBaskets_Unauthorized(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:55555/api/baskets", strings.NewReader(""))
	if assert.NoError(t, err) {
		// no authorization at all: 401 - unauthorized
		w := httptest.NewRecorder()
		GetBaskets(w, r, make(httprouter.Params, 0))
		assert.Equal(t, 401, w.Code, "wrong HTTP result code")

		// invalid master token: 401 - unauthorized
		r.Header.Add("Authorization", "123-wrong-token")
		w = httptest.NewRecorder()
		GetBaskets(w, r, make(httprouter.Params, 0))
		assert.Equal(t, 401, w.Code, "wrong HTTP result code")
	}
}

func TestGetStats(t *testing.T) {
	// create 3 baskets
	for i := 0; i < 3; i++ {
		basket := fmt.Sprintf("forstats0%v", i)
		r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
		if assert.NoError(t, err) {
			w := httptest.NewRecorder()
			ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
			CreateBasket(w, r, ps)
			assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		}
	}

	// get stats
	r, err := http.NewRequest("GET", "http://localhost:55555/api/stats", strings.NewReader(""))
	if assert.NoError(t, err) {
		r.Header.Add("Authorization", serverConfig.MasterToken)
		w := httptest.NewRecorder()
		GetStats(w, r, make(httprouter.Params, 0))
		// HTTP 200 - OK
		assert.Equal(t, 200, w.Code, "wrong HTTP result code")

		stats := new(DatabaseStats)
		err = json.Unmarshal(w.Body.Bytes(), stats)
		if assert.NoError(t, err) {
			// validate response
			assert.NotEmpty(t, stats.TopBasketsByDate, "top baskets are expected")
			assert.NotEmpty(t, stats.TopBasketsBySize, "top baskets are expected")
			assert.True(t, stats.BasketsCount > 0, "baskets count should be greater than 0")
			assert.True(t, stats.EmptyBasketsCount > 0, "empty baskets count should be greater than 0")
		}
	}
}

func TestGetStats_Unauthorized(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:55555/api/stats", strings.NewReader(""))
	if assert.NoError(t, err) {
		// no authorization at all: 401 - unauthorized
		w := httptest.NewRecorder()
		GetStats(w, r, make(httprouter.Params, 0))
		assert.Equal(t, 401, w.Code, "wrong HTTP result code")

		// invalid master token: 401 - unauthorized
		r.Header.Add("Authorization", "123-wrong-token")
		w = httptest.NewRecorder()
		GetStats(w, r, make(httprouter.Params, 0))
		assert.Equal(t, 401, w.Code, "wrong HTTP result code")
	}
}

func TestGetVersion(t *testing.T) {
	// get version
	r, err := http.NewRequest("GET", "http://localhost:55555/api/version", strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		GetVersion(w, r, make(httprouter.Params, 0))
		// HTTP 200 - OK
		assert.Equal(t, 200, w.Code, "wrong HTTP result code")

		ver := new(Version)
		err = json.Unmarshal(w.Body.Bytes(), ver)
		if assert.NoError(t, err) {
			// validate response
			assert.Equal(t, serviceName, ver.Name)
			assert.Equal(t, sourceCodeURL, ver.SourceCode)
			assert.NotEmpty(t, ver.Version, "version is expected")
			assert.NotEmpty(t, ver.Commit, "commit is expected")
			assert.NotEmpty(t, ver.CommitShort, "commit short is expected")
		}
	}
}

func TestGetBaskets_Query(t *testing.T) {
	// create 10 baskets
	for i := 0; i < 10; i++ {
		basket := fmt.Sprintf("names1%v", i)
		r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
		if assert.NoError(t, err) {
			w := httptest.NewRecorder()
			ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
			CreateBasket(w, r, ps)
			assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		}
	}

	// get names
	r, err := http.NewRequest("GET", "http://localhost:55555/api/baskets?q=names1", strings.NewReader(""))
	if assert.NoError(t, err) {
		r.Header.Add("Authorization", serverConfig.MasterToken)
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
		r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
		if assert.NoError(t, err) {
			w := httptest.NewRecorder()
			ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
			CreateBasket(w, r, ps)
			assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		}
	}

	// get names
	r, err := http.NewRequest("GET", "http://localhost:55555/api/baskets?max=5&skip=2", strings.NewReader(""))
	if assert.NoError(t, err) {
		r.Header.Add("Authorization", serverConfig.MasterToken)
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

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
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
			r, err = http.NewRequest("GET", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
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

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
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
			r, err = http.NewRequest("GET", "http://localhost:55555/api/baskets/"+basket+"?q=magic&in=headers", strings.NewReader(""))
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

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
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
			r, err = http.NewRequest("GET", "http://localhost:55555/api/baskets/"+basket+"?max=5&skip=5", strings.NewReader(""))
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

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
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
			r, err = http.NewRequest("DELETE", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				w = httptest.NewRecorder()
				ClearBasket(w, r, ps)
				// HTTP 204 - no content
				assert.Equal(t, 204, w.Code, "wrong HTTP result code")

				// get requests
				r, err = http.NewRequest("GET", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
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
						assert.Equal(t, 0, requests.Count, "wrong count of requests")
						assert.Equal(t, 25, requests.TotalCount, "wrong total count of requests")
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

func TestAcceptBasketRequests_BadRequest(t *testing.T) {
	basket := "accept03%20"
	req := createTestPOSTRequest("http://localhost:55555/"+basket, "my data", "text/plain")
	w := httptest.NewRecorder()
	AcceptBasketRequests(w, req)
	// HTTP 400 - Bad Request
	assert.Equal(t, 400, w.Code, "wrong HTTP result code")
	assert.Equal(t, "invalid basket name; the name does not match pattern: "+validBasketName.String()+"\n", w.Body.String(),
		"wrong error message")
}

func TestForwardToWeb(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:55555/", strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ForwardToWeb(w, r, make(httprouter.Params, 0))

		// validate response: 302 - Found
		assert.Equal(t, 302, w.Code, "wrong HTTP result code")
		assert.Equal(t, "/"+serviceUIPath, w.Header().Get("Location"), "wrong Location header")
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

func TestWebBasketsPage(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:55555/web/"+serviceOldAPIPath, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: serviceOldAPIPath})
		WebBasketPage(w, r, ps)

		// validate response: 200 - OK
		assert.Equal(t, 200, w.Code, "wrong HTTP result code")
		assert.Contains(t, w.Body.String(), "<title>Request Baskets - Administration</title>",
			"HTML index page with baskets is expected")
	}
}

func TestWebBasketPage_InvalidName(t *testing.T) {
	basket := ">>>"

	r, err := http.NewRequest("GET", "http://localhost:55555/web/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		WebBasketPage(w, r, ps)

		// validate response: 400 - Bad Request
		assert.Equal(t, 400, w.Code, "wrong HTTP result code")
		assert.Equal(t, "Basket name does not match pattern: "+validBasketName.String()+"\n", w.Body.String(),
			"wrong error message")
	}
}

func TestGetBasketResponse(t *testing.T) {
	basket := "response01"
	method := "GET"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("GET", "http://localhost:55555/api/baskets/"+basket+"/responses/"+method, strings.NewReader(""))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket},
					httprouter.Param{Key: "method", Value: method})
				w = httptest.NewRecorder()
				GetBasketResponse(w, r, ps)

				// validate response: 200 - OK
				assert.Equal(t, 200, w.Code, "wrong HTTP result code")
				assert.Equal(t, "application/json; charset=UTF-8", w.Header().Get("Content-Type"), "wrong Content-Type")

				// Default response: HTTP 200 - empty content
				response := new(ResponseConfig)
				err = json.Unmarshal(w.Body.Bytes(), response)
				if assert.NoError(t, err, "Failed to parse GetBasketResponse response") {
					assert.Equal(t, 200, response.Status, "wrong basket default response status")
					assert.Empty(t, response.Body, "empty response is expected")
					assert.Empty(t, response.Headers, "empty headers are expected")
					assert.False(t, response.IsTemplate, "wrong value of IsTemplate flag")
				}
			}
		}
	}
}

func TestGetBasketResponse_InvalidMethod(t *testing.T) {
	basket := "response02"
	method := "DEMO"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("GET", "http://localhost:55555/api/baskets/"+basket+"/responses/"+method, strings.NewReader(""))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket},
					httprouter.Param{Key: "method", Value: method})
				w = httptest.NewRecorder()
				GetBasketResponse(w, r, ps)

				// validate response: 400 - Bad Request
				assert.Equal(t, 400, w.Code, "wrong HTTP result code")
				assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"), "wrong Content-Type")
				assert.Contains(t, w.Body.String(), "unknown HTTP method: "+method, "wrong response message")
			}
		}
	}
}

func TestGetBasketResponse_Unauthorized(t *testing.T) {
	basket := "response03"
	method := "POST"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		r, err = http.NewRequest("GET", "http://localhost:55555/api/baskets/"+basket+"/responses/"+method, strings.NewReader(""))
		if assert.NoError(t, err) {
			ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket},
				httprouter.Param{Key: "method", Value: method})
			w = httptest.NewRecorder()
			GetBasketResponse(w, r, ps)

			// validate response: 401 - unauthorized
			assert.Equal(t, 401, w.Code, "wrong HTTP result code")
		}
	}
}

func TestUpdateBasketResponse(t *testing.T) {
	basket := "response04"
	method := "DELETE"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket+"/responses/"+method,
				strings.NewReader("{\"status\":404,\"body\":\"<error><code>404</code><message>Not Found</message></error>\",\"headers\":{\"Content-Type\":[\"application/xml\"]}}"))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket},
					httprouter.Param{Key: "method", Value: method})
				w = httptest.NewRecorder()

				UpdateBasketResponse(w, r, ps)

				// validate response: 204 - No Content
				assert.Equal(t, 204, w.Code, "wrong HTTP result code")

				// validate database update
				response := basketsDb.Get(basket).GetResponse(method)
				assert.Equal(t, 404, response.Status, "wrong response status")
				assert.Equal(t, "application/xml", response.Headers["Content-Type"][0], "wrong Content-Type header")
				assert.Equal(t, "<error><code>404</code><message>Not Found</message></error>", response.Body, "wrong response body")
				assert.False(t, response.IsTemplate, "wrong value of IsTemplate flag")
			}
		}
	}
}

func TestUpdateBasketResponse_InvalidMethod(t *testing.T) {
	basket := "response05"
	method := "WRONG"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket+"/responses/"+method,
				strings.NewReader("{\"status\":201,\"body\":\"{}\",\"headers\":{\"Content-Type\":[\"application/json\"]}}"))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket},
					httprouter.Param{Key: "method", Value: method})
				w = httptest.NewRecorder()

				UpdateBasketResponse(w, r, ps)

				// validate response: 400 - Bad Request
				assert.Equal(t, 400, w.Code, "wrong HTTP result code")
				assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"), "wrong Content-Type")
				assert.Contains(t, w.Body.String(), "unknown HTTP method: "+method, "wrong response message")

				// validate database is not updated
				assert.Nil(t, basketsDb.Get(basket).GetResponse(method), "response configuration is not expected")
			}
		}
	}
}

func TestUpdateBasketResponse_BrokenJson(t *testing.T) {
	basket := "response06"
	method := "PUT"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket+"/responses/"+method,
				strings.NewReader("<config><status>204</status></config>"))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket},
					httprouter.Param{Key: "method", Value: method})
				w = httptest.NewRecorder()

				UpdateBasketResponse(w, r, ps)

				// validate response: 400 - Bad Request
				assert.Equal(t, 400, w.Code, "wrong HTTP result code")
				assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"), "wrong Content-Type")
				assert.Contains(t, w.Body.String(), "invalid character '<'", "wrong response message")

				// validate database is not updated
				assert.Nil(t, basketsDb.Get(basket).GetResponse(method), "response configuration is not expected")
			}
		}
	}
}

func TestUpdateBasketResponse_InvalidConfig(t *testing.T) {
	basket := "response07"
	method := "OPTIONS"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket+"/responses/"+method,
				strings.NewReader("{\"status\":20}"))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket},
					httprouter.Param{Key: "method", Value: method})
				w = httptest.NewRecorder()

				UpdateBasketResponse(w, r, ps)

				// validate response: 422 - Unprocessable Entity
				assert.Equal(t, 422, w.Code, "wrong HTTP result code")
				assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"), "wrong Content-Type")
				assert.Contains(t, w.Body.String(), "invalid HTTP status of response: 20", "wrong response message")

				// validate database is not updated
				assert.Nil(t, basketsDb.Get(basket).GetResponse(method), "response configuration is not expected")
			}
		}
	}
}

func TestUpdateBasketResponse_InvalidTemplate(t *testing.T) {
	basket := "response08"
	method := "GET"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket+"/responses/"+method,
				strings.NewReader("{\"status\":200,\"body\":\"data: {{data}}\",\"is_template\":true}"))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket},
					httprouter.Param{Key: "method", Value: method})
				w = httptest.NewRecorder()

				UpdateBasketResponse(w, r, ps)

				// validate response: 422 - Unprocessable Entity
				assert.Equal(t, 422, w.Code, "wrong HTTP result code")
				assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"), "wrong Content-Type")
				assert.Contains(t, w.Body.String(), "error in body template: body:1: function \"data\" not defined", "wrong response message")

				// validate database is not updated
				assert.Nil(t, basketsDb.Get(basket).GetResponse(method), "response configuration is not expected")
			}
		}
	}
}

func TestUpdateBasketResponse_NotModified(t *testing.T) {
	basket := "response09"
	method := "GET"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket+"/responses/"+method, strings.NewReader(""))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket},
					httprouter.Param{Key: "method", Value: method})
				w = httptest.NewRecorder()

				UpdateBasketResponse(w, r, ps)

				// validate response: 304 - Not Modified
				assert.Equal(t, 304, w.Code, "wrong HTTP result code")

				// validate database is not updated
				assert.Nil(t, basketsDb.Get(basket).GetResponse(method), "response configuration is not expected")
			}
		}
	}
}

func TestUpdateBasketResponse_ReadTimeout(t *testing.T) {
	basket := "response10"
	method := "PATCH"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket+"/responses/"+method,
				iotest.TimeoutReader(strings.NewReader("object is patched")))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket},
					httprouter.Param{Key: "method", Value: method})
				w = httptest.NewRecorder()

				UpdateBasketResponse(w, r, ps)

				// validate response: 500 - Internal Server Error
				assert.Equal(t, 500, w.Code, "wrong HTTP result code")

				// validate database is not updated
				assert.Nil(t, basketsDb.Get(basket).GetResponse(method), "response configuration is not expected")
			}
		}
	}
}

func TestAcceptBasketRequests_CustomResponse(t *testing.T) {
	basket := "accept03"
	method := "POST"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket+"/responses/"+method,
				strings.NewReader("{\"status\":201,\"body\":\"successfully created\",\"headers\":{"+
					"\"Location\":[\"http://localhost:55555/id/1234\"],\"X-Rate-Limit\":[\"10\",\"1000\"]}}"))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket},
					httprouter.Param{Key: "method", Value: method})
				w = httptest.NewRecorder()

				UpdateBasketResponse(w, r, ps)

				// validate response: 204 - No Content
				assert.Equal(t, 204, w.Code, "wrong HTTP result code")

				r, err = http.NewRequest(method, "http://localhost:55555/"+basket, strings.NewReader("test"))
				if assert.NoError(t, err) {
					w = httptest.NewRecorder()

					AcceptBasketRequests(w, r)

					// validate expected response
					assert.Equal(t, 201, w.Code, "wrong HTTP response code")
					assert.Equal(t, "successfully created", w.Body.String(), "wrong HTTP response body")
					assert.Equal(t, "http://localhost:55555/id/1234", w.Header().Get("Location"), "wrong HTTP header")
				}
			}
		}
	}
}

func TestAcceptBasketRequests_TemplateResponse(t *testing.T) {
	basket := "accept04"
	method := "GET"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// get auth token
		auth := new(BasketAuth)
		err = json.Unmarshal(w.Body.Bytes(), auth)
		if assert.NoError(t, err, "Failed to parse CreateBasket response") {
			r, err = http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket+"/responses/"+method,
				strings.NewReader("{\"status\":200,\"body\":\"hello {{range .name}}{{.}} {{end}}\",\"is_template\":true}"))

			if assert.NoError(t, err) {
				r.Header.Add("Authorization", auth.Token)
				ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket},
					httprouter.Param{Key: "method", Value: method})
				w = httptest.NewRecorder()

				UpdateBasketResponse(w, r, ps)

				// validate response: 204 - No Content
				assert.Equal(t, 204, w.Code, "wrong HTTP result code")

				r, err = http.NewRequest(method, "http://localhost:55555/"+basket+"?name=Adam&name=Dan", strings.NewReader("test"))
				if assert.NoError(t, err) {
					w = httptest.NewRecorder()

					AcceptBasketRequests(w, r)

					// validate expected response
					assert.Equal(t, 200, w.Code, "wrong HTTP response code")
					assert.Equal(t, "hello Adam Dan ", w.Body.String(), "wrong HTTP response body")
				}
			}
		}
	}
}

func TestAcceptBasketRequests_WithForwardInsecure(t *testing.T) {
	basket := "accept05"
	method := "PUT"

	// Test HTTP server
	var forwardedData *RequestData
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedData = ToRequestData(r)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Config to forward requests to test HTTP server (also enable expanding URL)
	forwardURL := ts.URL + "/notify?captured_at=" + basket

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket,
		strings.NewReader("{\"forward_url\":\""+forwardURL+"\",\"insecure_tls\":true,\"capacity\":200}"))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// send request and validate forwarding
		r, err = http.NewRequest(method, "http://localhost:55555/"+basket+"/articles/123?name=Adam&age=33",
			strings.NewReader("new text from Adam"))
		if assert.NoError(t, err) {
			w = httptest.NewRecorder()
			AcceptBasketRequests(w, r)
			// validate expected response
			assert.Equal(t, 200, w.Code, "wrong HTTP response code")
			time.Sleep(100 * time.Millisecond)

			// validate forwarded request
			assert.Equal(t, "/notify", forwardedData.Path, "wrong request path")
			assert.Equal(t, "captured_at="+basket+"&name=Adam&age=33", forwardedData.Query, "wrong request query")
			assert.Equal(t, "new text from Adam", forwardedData.Body, "wrong request body")
			assert.Equal(t, method, forwardedData.Method, "wrong request method")
		}
	}
}

func TestAcceptBasketRequests_WithForwardExpand(t *testing.T) {
	basket := "accept06"
	method := "DELETE"

	// Test HTTP server
	var forwardedData *RequestData
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedData = ToRequestData(r)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Config to forward requests to test HTTP server (also enable expanding URL)
	forwardURL := ts.URL + "/service?from=" + basket

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket,
		strings.NewReader("{\"forward_url\":\""+forwardURL+"\",\"expand_path\":true,\"capacity\":200}"))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		r, err = http.NewRequest(method, "http://localhost:55555/"+basket+"/articles/123?sig=abcdge3276542",
			strings.NewReader(""))
		r.Header.Add("X-Client", "Java/1.8")

		if assert.NoError(t, err) {
			w = httptest.NewRecorder()
			AcceptBasketRequests(w, r)
			// validate expected response
			assert.Equal(t, 200, w.Code, "wrong HTTP response code")
			time.Sleep(100 * time.Millisecond)

			// validate forwarded request
			assert.Equal(t, "/service/articles/123", forwardedData.Path, "wrong request path")
			assert.Equal(t, "from="+basket+"&sig=abcdge3276542", forwardedData.Query, "wrong request query")
			assert.Equal(t, "", forwardedData.Body, "wrong request body")
			assert.Equal(t, method, forwardedData.Method, "wrong request method")
			assert.Equal(t, "Java/1.8", forwardedData.Header.Get("X-Client"), "wrong request header")
		}
	}
}

func TestAcceptBasketRequests_WithProxyResponse(t *testing.T) {
	basket := "accept07"
	method := "DELETE"

	// Test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("server test response"))
	}))
	defer ts.Close()

	// Config to forward requests to test HTTP server (also enable expanding URL)
	forwardURL := ts.URL + "/service?from=" + basket

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket,
		strings.NewReader("{\"forward_url\":\""+forwardURL+"\",\"expand_path\":true,\"capacity\":200,\"proxy_response\":true}"))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		r, err = http.NewRequest(method, "http://localhost:55555/"+basket+"/articles/123?sig=abcdge3276542",
			strings.NewReader(""))
		r.Header.Add("X-Client", "Java/1.8")

		if assert.NoError(t, err) {
			w = httptest.NewRecorder()
			AcceptBasketRequests(w, r)

			// read response body
			responseBody, err := ioutil.ReadAll(w.Body)
			assert.NoError(t, err)

			// validate expected response
			assert.Equal(t, 202, w.Code, "wrong HTTP response code")
			assert.Equal(t, "server test response", string(responseBody), "wrong response body")
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func TestAcceptBasketRequests_WithForward_BadGateway(t *testing.T) {
	basket := "accept08"
	method := "GET"

	// assuming that nothing is running at port 55556
	forwardURL := "http://localhost:55556/notify"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket,
		strings.NewReader("{\"forward_url\":\""+forwardURL+"\",\"capacity\":200}"))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// send request and validate forwarding
		r, err = http.NewRequest(method, "http://localhost:55555/"+basket+"/faile_to_forward", strings.NewReader(""))
		if assert.NoError(t, err) {
			w = httptest.NewRecorder()
			AcceptBasketRequests(w, r)

			// validate expected response: forwarding errors are not exposed unless ForwardResponse is enabled
			assert.Equal(t, 200, w.Code, "wrong HTTP response code")
			assert.Equal(t, "", w.Body.String(), "wrong HTTP response body")
		}
	}
}

func TestAcceptBasketRequests_WithProxyResponse_BadGateway(t *testing.T) {
	basket := "accept09"
	method := "POST"

	// assuming that nothing is running at port 55556
	forwardURL := "http://localhost:55556/notify"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket,
		strings.NewReader("{\"forward_url\":\""+forwardURL+"\",\"proxy_response\":true,\"capacity\":20}"))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// send request and validate forwarding
		r, err = http.NewRequest(method, "http://localhost:55555/"+basket+"/faile_to_forward", strings.NewReader(""))
		if assert.NoError(t, err) {
			w = httptest.NewRecorder()
			AcceptBasketRequests(w, r)

			// validate expected response
			assert.Equal(t, 502, w.Code, "wrong HTTP response code")
			assert.Contains(t, w.Body.String(), "Failed to forward request", "wrong HTTP response body")
			assert.Contains(t, w.Body.String(), forwardURL, "wrong HTTP response body")
			assert.Contains(t, w.Body.String(), "connection refused", "wrong HTTP response body")
		}
	}
}

func TestAcceptBasketRequests_WithForward_InternalServerError(t *testing.T) {
	basket := "accept10"
	method := "PUT"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// set invalid forward URL directly
		b := basketsDb.Get(basket)
		b.Update(BasketConfig{Capacity: 20, ForwardURL: "qwert"})

		// send request and validate forwarding
		r, err = http.NewRequest(method, "http://localhost:55555/"+basket+"/internal_error", strings.NewReader("abc"))
		if assert.NoError(t, err) {
			w = httptest.NewRecorder()
			AcceptBasketRequests(w, r)

			// validate expected response: forwarding errors are not exposed unless ForwardResponse is enabled
			assert.Equal(t, 200, w.Code, "wrong HTTP response code")
			assert.Equal(t, "", w.Body.String(), "wrong HTTP response body")
		}
	}
}

func TestAcceptBasketRequests_WithProxyResponse_InternalServerError(t *testing.T) {
	basket := "accept11"
	method := "PATCH"

	r, err := http.NewRequest("POST", "http://localhost:55555/api/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		w := httptest.NewRecorder()

		CreateBasket(w, r, ps)
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")

		// set invalid forward URL directly
		b := basketsDb.Get(basket)
		b.Update(BasketConfig{Capacity: 20, ForwardURL: "qwert", ProxyResponse: true})

		// send request and validate forwarding
		r, err = http.NewRequest(method, "http://localhost:55555/"+basket+"/internal_error", strings.NewReader("abc"))
		if assert.NoError(t, err) {
			w = httptest.NewRecorder()
			AcceptBasketRequests(w, r)

			// validate expected response
			assert.Equal(t, 500, w.Code, "wrong HTTP response code")
			assert.Contains(t, w.Body.String(), "invalid forward URL: qwert", "wrong HTTP response body")
			assert.Contains(t, w.Body.String(), "invalid URI for request", "wrong HTTP response body")
		}
	}
}

func TestGetBasketNameOfAcceptedRequest_NoPrefix_Valid(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:55555/basket200", strings.NewReader(""))
	if assert.NoError(t, err) {
		name, pubErr, err := getBasketNameOfAcceptedRequest(r, "")
		assert.Equal(t, "basket200", name, "unexpected basket name")
		assert.Empty(t, pubErr)
		assert.Nil(t, err)
	}
}

func TestGetBasketNameOfAcceptedRequest_NoPrefix_ValidWithSubpath(t *testing.T) {
	r, err := http.NewRequest("DELETE", "http://localhost:55555/basket210/api/users/123", strings.NewReader(""))
	if assert.NoError(t, err) {
		name, pubErr, err := getBasketNameOfAcceptedRequest(r, "")
		assert.Equal(t, "basket210", name, "unexpected basket name")
		assert.Empty(t, pubErr)
		assert.Nil(t, err)
	}
}

func TestGetBasketNameOfAcceptedRequest_NoPrefix_Invalid(t *testing.T) {
	r, err := http.NewRequest("PUT", "http://localhost:55555/basket~220/objects/404", strings.NewReader("{}"))
	if assert.NoError(t, err) {
		name, pubErr, err := getBasketNameOfAcceptedRequest(r, "")
		assert.Empty(t, name, "basket name is invalid, hence unexpected")
		assert.Equal(t, "invalid basket name; the name does not match pattern: "+basketNamePattern, pubErr)
		if assert.NotNil(t, err) {
			assert.Equal(t, "invalid basket name; the name does not match pattern: "+
				basketNamePattern+"; request: PUT /basket~220/objects/404", err.Error())
		}
	}
}

func TestGetBasketNameOfAcceptedRequest_WithPrefix_Valid(t *testing.T) {
	r, err := http.NewRequest("GET", "http://localhost:55555/abc/basket300", strings.NewReader(""))
	if assert.NoError(t, err) {
		name, pubErr, err := getBasketNameOfAcceptedRequest(r, "/abc")
		assert.Equal(t, "basket300", name, "unexpected basket name")
		assert.Empty(t, pubErr)
		assert.Nil(t, err)
	}
}

func TestGetBasketNameOfAcceptedRequest_WithPrefix_ValidWithSubpath(t *testing.T) {
	r, err := http.NewRequest("PATCH", "http://localhost:55555/xyz/basket310/api/users/123", strings.NewReader("{}"))
	if assert.NoError(t, err) {
		name, pubErr, err := getBasketNameOfAcceptedRequest(r, "/xyz")
		assert.Equal(t, "basket310", name, "unexpected basket name")
		assert.Empty(t, pubErr)
		assert.Nil(t, err)
	}
}

func TestGetBasketNameOfAcceptedRequest_WithPrefix_OutsideOfContext(t *testing.T) {
	r, err := http.NewRequest("POST", "http://localhost:55555/api/objects", strings.NewReader("{}"))
	if assert.NoError(t, err) {
		name, pubErr, err := getBasketNameOfAcceptedRequest(r, "/baskets")
		assert.Empty(t, name, "URL is out of context, hence no basket name is expected")
		assert.Equal(t, "incoming request is outside of configured path prefix: /baskets", pubErr)
		if assert.NotNil(t, err) {
			assert.Equal(t, "incoming request is outside of configured path prefix: /baskets"+
				"; request: POST /api/objects", err.Error())
		}
	}
}

func TestSanitizeForLog(t *testing.T) {
	assert.Equal(t, "basket2346", sanitizeForLog("basket2346"), "unexpected result of sanitizing")
	assert.Equal(t, "abc~!@#$%09381", sanitizeForLog("abc~!@#$%09381"), "unexpected result of sanitizing")
	assert.Equal(t, "new line^n injection", sanitizeForLog("new line\n injection"), "unexpected result of sanitizing")
	assert.Equal(t, "another^rnew line", sanitizeForLog("another\rnew line"), "unexpected result of sanitizing")
	assert.Equal(t, "multi-^n^r^n^r^rmulti-^nmulti-^r^nlines", sanitizeForLog("multi-\n\r\n\r\rmulti-\nmulti-\r\nlines"),
		"unexpected result of sanitizing")
}

func TestCreateTemplateData(t *testing.T) {
	r := &RequestData{
		Body: "{\n    \"data\": {\n        \"authorizationId\": \"4bc09f83-19d3-41ca-b6ee-68d5fb293ae7\"\n    },\n    \"eventName\": \"request\",\n    \"eventType\": \"authorization\"\n}",
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Query: "name=ming&test=aa",
	}
	templateData := createTemplateData(r)
	templateResponse := `
{
    "authorizationId":"{{.body.data.authorizationId}}",
    "query":"{{index .query.name 0}}",
    "query-back-compatibility":"{{index .name 0}}",
    "responseCode":"00"
}
`
	var result bytes.Buffer
	tmpl, _ := template.New("testTemplateResponse").Parse(templateResponse)
	_ = tmpl.Execute(&result, templateData)
	assert.Equal(t, `
{
    "authorizationId":"4bc09f83-19d3-41ca-b6ee-68d5fb293ae7",
    "query":"ming",
    "query-back-compatibility":"ming",
    "responseCode":"00"
}
`, result.String())
}
