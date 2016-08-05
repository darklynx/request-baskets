package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

var validBasketName = regexp.MustCompile(BASKET_NAME)
var indexPage = template.Must(template.New("index").Parse(INDEX_HTML))
var basketPage = template.Must(template.New("basket").Parse(BASKET_HTML))

// writeJson writes JSON content to HTTP response
func writeJson(w http.ResponseWriter, status int, json []byte, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(status)
		w.Write(json)
	}
}

// parseInt parses integer parameter from HTTP request query
func parseInt(value string, defaultValue int) int {
	if len(value) > 0 {
		i, err := strconv.Atoi(value)
		if err == nil {
			return i
		}
	}

	return defaultValue
}

// getPage retrieves page settings from HTTP request query params
func getPage(values url.Values) (int, int) {
	max := parseInt(values.Get("max"), serverConfig.PageSize)
	skip := parseInt(values.Get("skip"), 0)

	return max, skip
}

// getAndAuthBasket retrieves basket by name from HTTP request path and authorize access to the basket object
func getAndAuthBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (string, Basket) {
	name := ps.ByName("basket")
	basket := basketsDb.Get(name)
	if basket != nil {
		// maybe custom header, e.g. basket_key, basket_token
		token := r.Header.Get("Authorization")
		if basket.Authorize(token) || token == serverConfig.MasterToken {
			return name, basket
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}

	return "", nil
}

// validateBasketConfig validates basket configuration
func validateBasketConfig(config *BasketConfig) error {
	// validate Capacity
	if config.Capacity < 1 {
		return fmt.Errorf("Capacity should be a positive number, but was %d", config.Capacity)
	}

	if config.Capacity > serverConfig.MaxCapacity {
		return fmt.Errorf("Capacity may not be greater than %d", serverConfig.MaxCapacity)
	}

	// validate URL
	if len(config.ForwardUrl) > 0 {
		if _, err := url.ParseRequestURI(config.ForwardUrl); err != nil {
			return err
		}
	}

	return nil
}

// GetBaskets handles HTTP request to get registered baskets
func GetBaskets(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	values := r.URL.Query()
	query := values.Get("q")
	if len(query) > 0 {
		// Find names
		max, skip := getPage(values)
		json, err := json.Marshal(basketsDb.FindNames(query, max, skip))
		writeJson(w, http.StatusOK, json, err)
	} else {
		// Get basket names page
		json, err := json.Marshal(basketsDb.GetNames(getPage(values)))
		writeJson(w, http.StatusOK, json, err)
	}
}

// GetBasket handles HTTP request to get basket configuration
func GetBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAndAuthBasket(w, r, ps); basket != nil {
		json, err := json.Marshal(basket.Config())
		writeJson(w, http.StatusOK, json, err)
	}
}

// CreateBasket handles HTTP request to create a new basket
func CreateBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	name := ps.ByName("basket")
	if name == BASKETS_ROOT || name == WEB_ROOT {
		http.Error(w, "Basket name may not clash with system path: "+name, http.StatusForbidden)
		return
	}
	if !validBasketName.MatchString(name) {
		http.Error(w, "Basket name does not match pattern: "+validBasketName.String(), http.StatusBadRequest)
		return
	}

	log.Printf("[info] creating basket: %s", name)

	// read config (max 2 kB)
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 2048))
	r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// default config
	config := BasketConfig{ForwardUrl: "", Capacity: serverConfig.InitCapacity}
	if len(body) > 0 {
		if err = json.Unmarshal(body, &config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = validateBasketConfig(&config); err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
	}

	auth, err := basketsDb.Create(name, config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
	} else {
		json, err := json.Marshal(auth)
		writeJson(w, http.StatusCreated, json, err)
	}
}

// UpdateBasket handles HTTP request to update basket configuration
func UpdateBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAndAuthBasket(w, r, ps); basket != nil {
		// read config (max 2 kB)
		body, err := ioutil.ReadAll(io.LimitReader(r.Body, 2048))
		r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else if len(body) > 0 {
			// get current config
			config := basket.Config()
			if err = json.Unmarshal(body, &config); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err = validateBasketConfig(&config); err != nil {
				http.Error(w, err.Error(), 422)
				return
			}

			basket.Update(config)

			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusNotModified)
		}
	}
}

// DeleteBasket handles HTTP request to delete basket
func DeleteBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if name, basket := getAndAuthBasket(w, r, ps); basket != nil {
		log.Printf("[info] deleting basket: %s", name)

		basketsDb.Delete(name)
		w.WriteHeader(http.StatusNoContent)
	}
}

// GetBasketRequests handles HTTP request to get requests collected by basket
func GetBasketRequests(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAndAuthBasket(w, r, ps); basket != nil {
		values := r.URL.Query()
		query := values.Get("q")
		if len(query) > 0 {
			// Find requests
			max, skip := getPage(values)
			json, err := json.Marshal(basket.FindRequests(query, values.Get("in"), max, skip))
			writeJson(w, http.StatusOK, json, err)
		} else {
			// Get requests page
			json, err := json.Marshal(basket.GetRequests(getPage(values)))
			writeJson(w, http.StatusOK, json, err)
		}
	}
}

// ClearBasket handles HTTP request to delete all requests collected by basket
func ClearBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAndAuthBasket(w, r, ps); basket != nil {
		basket.Clear()
		w.WriteHeader(http.StatusNoContent)
	}
}

// ForwardToWeb handels HTTP forwarding to /web
func ForwardToWeb(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	http.Redirect(w, r, "/"+WEB_ROOT, 302)
}

// WebIndexPage handles HTTP request to render index page
func WebIndexPage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	indexPage.Execute(w, "")
}

// WebBasketPage handles HTTP request to render basket details page
func WebBasketPage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	name := ps.ByName("basket")
	if !validBasketName.MatchString(name) {
		http.Error(w, "Basket name does not match pattern: "+validBasketName.String(), http.StatusBadRequest)
		return
	}
	basketPage.Execute(w, name)
}

// AcceptBasketRequests accepts and handles HTTP requests passed to different baskets
func AcceptBasketRequests(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	name := parts[1]
	basket := basketsDb.Get(name)
	if basket != nil {
		request := basket.Add(r)
		w.WriteHeader(http.StatusOK)

		if len(basket.Config().ForwardUrl) > 0 {
			go request.Forward(basket.Config(), name)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
