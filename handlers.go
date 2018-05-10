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

var validBasketName = regexp.MustCompile(basketNamePattern)
var defaultResponse = ResponseConfig{Status: 200, Headers: http.Header{}, IsTemplate: false}
var basketPageTemplate = template.Must(template.New("basket").Parse(basketPageContentTemplate))

// writeJSON writes JSON content to HTTP response
func writeJSON(w http.ResponseWriter, status int, json []byte, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(status)
		w.Write(json)
	}
}

// parseInt parses integer parameter from HTTP request query
func parseInt(value string, min int, max int, defaultValue int) int {
	if len(value) > 0 {
		if i, err := strconv.Atoi(value); err == nil {
			switch {
			case i < min:
				return min
			case i > max:
				return max
			default:
				return i
			}
		}
	}

	return defaultValue
}

// getPage retrieves page settings from HTTP request query params
func getPage(values url.Values) (int, int) {
	max := parseInt(values.Get("max"), 1, serverConfig.PageSize*10, serverConfig.PageSize)
	skip := parseInt(values.Get("skip"), 0, serverConfig.MaxCapacity, 0)

	return max, skip
}

// getAndAuthBasket retrieves basket by name from HTTP request path and authorize access to the basket object
func getAndAuthBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (string, Basket) {
	name := ps.ByName("basket")
	if basket := basketsDb.Get(name); basket != nil {
		// maybe custom header, e.g. basket_key, basket_token
		if token := r.Header.Get("Authorization"); basket.Authorize(token) || token == serverConfig.MasterToken {
			return name, basket
		}
		w.WriteHeader(http.StatusUnauthorized)
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
	if len(config.ForwardURL) > 0 {
		if _, err := url.ParseRequestURI(config.ForwardURL); err != nil {
			return err
		}
	}

	return nil
}

// validateResponseConfig validates basket response configuration
func validateResponseConfig(config *ResponseConfig) error {
	// validate status
	if config.Status < 100 || config.Status >= 600 {
		return fmt.Errorf("Invalid HTTP status of response: %d", config.Status)
	}

	// validate template
	if config.IsTemplate && len(config.Body) > 0 {
		if _, err := template.New("body").Parse(config.Body); err != nil {
			return fmt.Errorf("Error in body %s", err)
		}
	}

	return nil
}

// getValidMethod retrieves mathod name from HTTP request path and validates it
func getValidMethod(ps httprouter.Params) (string, error) {
	method := strings.ToUpper(ps.ByName("method"))

	// valid HTTP methods
	switch method {
	case http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace:
		return method, nil
	}

	return method, fmt.Errorf("Unknown HTTP method: %s", method)
}

// GetBaskets handles HTTP request to get registered baskets
func GetBaskets(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if r.Header.Get("Authorization") != serverConfig.MasterToken {
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		values := r.URL.Query()
		if query := values.Get("q"); len(query) > 0 {
			// Find names
			max, skip := getPage(values)
			json, err := json.Marshal(basketsDb.FindNames(query, max, skip))
			writeJSON(w, http.StatusOK, json, err)
		} else {
			// Get basket names page
			json, err := json.Marshal(basketsDb.GetNames(getPage(values)))
			writeJSON(w, http.StatusOK, json, err)
		}
	}
}

// GetBasket handles HTTP request to get basket configuration
func GetBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAndAuthBasket(w, r, ps); basket != nil {
		json, err := json.Marshal(basket.Config())
		writeJSON(w, http.StatusOK, json, err)
	}
}

// CreateBasket handles HTTP request to create a new basket
func CreateBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	name := ps.ByName("basket")
	if name == serviceAPIPath || name == serviceUIPath {
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// default config
	config := BasketConfig{ForwardURL: "", Capacity: serverConfig.InitCapacity}
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
		writeJSON(w, http.StatusCreated, json, err)
	}
}

// UpdateBasket handles HTTP request to update basket configuration
func UpdateBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAndAuthBasket(w, r, ps); basket != nil {
		// read config (max 2 kB)
		body, err := ioutil.ReadAll(io.LimitReader(r.Body, 2048))
		r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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

// GetBasketResponse handles HTTP request to get basket response configuration
func GetBasketResponse(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAndAuthBasket(w, r, ps); basket != nil {
		method, errm := getValidMethod(ps)
		if errm != nil {
			http.Error(w, errm.Error(), http.StatusBadRequest)
		} else {
			response := basket.GetResponse(method)
			if response == nil {
				response = &defaultResponse
			}

			json, err := json.Marshal(response)
			writeJSON(w, http.StatusOK, json, err)
		}
	}
}

// UpdateBasketResponse handles HTTP request to update basket response configuration
func UpdateBasketResponse(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAndAuthBasket(w, r, ps); basket != nil {
		method, errm := getValidMethod(ps)
		if errm != nil {
			http.Error(w, errm.Error(), http.StatusBadRequest)
		} else {
			// read response (max 64 kB)
			body, err := ioutil.ReadAll(io.LimitReader(r.Body, 64*1024))
			r.Body.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else if len(body) > 0 {
				// get current config
				response := ResponseConfig{Status: defaultResponse.Status, IsTemplate: false}
				if err = json.Unmarshal(body, &response); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if err = validateResponseConfig(&response); err != nil {
					http.Error(w, err.Error(), 422)
					return
				}

				basket.SetResponse(method, response)
				w.WriteHeader(http.StatusNoContent)
			} else {
				w.WriteHeader(http.StatusNotModified)
			}
		}
	}
}

// GetBasketRequests handles HTTP request to get requests collected by basket
func GetBasketRequests(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAndAuthBasket(w, r, ps); basket != nil {
		values := r.URL.Query()
		if query := values.Get("q"); len(query) > 0 {
			// Find requests
			max, skip := getPage(values)
			json, err := json.Marshal(basket.FindRequests(query, values.Get("in"), max, skip))
			writeJSON(w, http.StatusOK, json, err)
		} else {
			// Get requests page
			json, err := json.Marshal(basket.GetRequests(getPage(values)))
			writeJSON(w, http.StatusOK, json, err)
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
	http.Redirect(w, r, "/"+serviceUIPath, http.StatusFound)
}

// WebIndexPage handles HTTP request to render index page
func WebIndexPage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexPageContent)
}

// WebBasketPage handles HTTP request to render basket details page
func WebBasketPage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if name := ps.ByName("basket"); validBasketName.MatchString(name) {
		switch name {
		case serviceAPIPath:
			// admin page to access all baskets
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(basketsPageContent)
		default:
			basketPageTemplate.Execute(w, name)
		}
	} else {
		http.Error(w, "Basket name does not match pattern: "+validBasketName.String(), http.StatusBadRequest)
	}
}

// AcceptBasketRequests accepts and handles HTTP requests passed to different baskets
func AcceptBasketRequests(w http.ResponseWriter, r *http.Request) {
	name := strings.Split(r.URL.Path, "/")[1]
	if basket := basketsDb.Get(name); basket != nil {
		request := basket.Add(r)

		responseConfig := basket.GetResponse(r.Method)
		if responseConfig == nil {
			responseConfig = &defaultResponse
		}

		// forward request in separate thread
		config := basket.Config()
		if len(config.ForwardURL) > 0 && r.Header.Get(DoNotForwardHeader) != "1" {
			client := httpClient
			if config.InsecureTLS {
				client = httpInsecureClient
			}

			if config.ProxyResponse {
				response := request.Forward(client, config, name)
				defer response.Body.Close()

				body, err := ioutil.ReadAll(response.Body)
				if err != nil {
					http.Error(w, "Error in "+err.Error(), http.StatusInternalServerError)
				}

				responseConfig.Headers = response.Header
				responseConfig.Status = response.StatusCode
				responseConfig.Body = string(body)
			} else {
				go request.Forward(client, config, name)
			}
		}

		// headers
		for k, v := range responseConfig.Headers {
			w.Header()[k] = v
		}
		// body
		if responseConfig.IsTemplate && len(responseConfig.Body) > 0 {
			// template
			t, err := template.New(name + "-" + r.Method).Parse(responseConfig.Body)
			if err != nil {
				// invalid template
				http.Error(w, "Error in "+err.Error(), http.StatusInternalServerError)
			} else {
				// status
				w.WriteHeader(responseConfig.Status)
				// templated body
				t.Execute(w, r.URL.Query())
			}
		} else {
			// status
			w.WriteHeader(responseConfig.Status)
			// plain body
			w.Write([]byte(responseConfig.Body))
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
