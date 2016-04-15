package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func testsSetup() {
	// Initialize global HTTP clients
	httpClient = new(http.Client)
	httpInsecureClient = new(http.Client)

	// Global config
	serverConfig = CreateConfig()
	// Global DB (in memory)
	basketsDb = createBasketsDatabase()
}

func testsShutdown() {
	// Release global DB
	basketsDb.Release()
}

func TestMain(m *testing.M) {
	testsSetup()
	code := m.Run()
	testsShutdown()
	os.Exit(code)
}

func TestCreateBasket(t *testing.T) {
	basket := "test001"

	r, err := http.NewRequest("POST", "http://localhost:55555/baskets/"+basket, strings.NewReader(""))
	if assert.NoError(t, err) {
		w := httptest.NewRecorder()
		ps := append(make(httprouter.Params, 0), httprouter.Param{Key: "basket", Value: basket})
		CreateBasket(w, r, ps)

		// validate response
		assert.Equal(t, 201, w.Code, "wrong HTTP result code")
		assert.Equal(t, "application/json; charset=UTF-8", w.Header().Get("Content-Type"), "wrong Content-Type")
		assert.Contains(t, w.Body.String(), "\"token\"", "JSON response with token is expected")

		// validate database
		assert.NotNil(t, basketsDb.Get(basket), "basket '%v' should be created", basket)
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
