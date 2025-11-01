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

func createTemplateData(r *RequestData) map[string]interface{} {
	data := make(map[string]interface{})

	// Parse query parameters from the Query string
	queryValues, err := url.ParseQuery(r.Query)
	if err != nil {
		// If parsing fails, create empty url.Values
		queryValues = make(url.Values)
	}

	// Add query parameters (maintain backward compatibility)
	for k, v := range queryValues {
		data[k] = v
	}

	data["query"] = queryValues
	data["headers"] = r.Header
	data["method"] = r.Method

	// Add request body and try to parse JSON
	if len(r.Body) > 0 && r.Header.Get("Content-Type") == "application/json" {
		// Try to parse JSON body
		var jsonData interface{}
		if err := json.Unmarshal([]byte(r.Body), &jsonData); err == nil {
			data["body"] = jsonData
		}
	}

	return data
}

const (
	ModePublic     = "public"
	ModeRestricted = "restricted"
)

var validBasketName = regexp.MustCompile(basketNamePattern)
var defaultResponse = ResponseConfig{Status: http.StatusOK, Headers: http.Header{}, IsTemplate: false}
var indexPageTemplate = template.Must(template.New("index").Parse(indexPageContentTemplate))
var basketPageTemplate = template.Must(template.New("basket").Parse(basketPageContentTemplate))
var basketsPageTemplate = template.Must(template.New("baskets").Parse(basketsPageContentTemplate))

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

// getAuthorizedBasket fetches basket details by name and authorizes the access to this basket, returns nil in case of failure
func getAuthorizedBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params, config *ServerConfig) (string, Basket) {
	name := ps.ByName("basket")
	if !validBasketName.MatchString(name) {
		http.Error(w, "invalid basket name; the name does not match pattern: "+validBasketName.String(), http.StatusBadRequest)
	} else if basket := basketsDb.Get(name); basket != nil {
		// maybe custom header, e.g. basket_key, basket_token
		if token := r.Header.Get("Authorization"); basket.Authorize(token) || token == config.MasterToken {
			return name, basket
		}
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}

	return "", nil
}

// authorizeRequest helps to authorize requests for restricted end-points and returns true in case of successful authorization
// publicAPI requires no authorization unless the server mode is set to "restricted"
func authorizeRequest(w http.ResponseWriter, r *http.Request, publicAPI bool, config *ServerConfig) bool {
	if publicAPI && config.Mode != ModeRestricted {
		return true
	}

	if r.Header.Get("Authorization") == serverConfig.MasterToken {
		return true
	}

	w.WriteHeader(http.StatusUnauthorized)
	return false
}

// validateBasketConfig validates basket configuration
func validateBasketConfig(config *BasketConfig) error {
	// validate Capacity
	if config.Capacity < 1 {
		return fmt.Errorf("capacity should be a positive number, but was %d", config.Capacity)
	}

	if config.Capacity > serverConfig.MaxCapacity {
		return fmt.Errorf("capacity may not be greater than %d", serverConfig.MaxCapacity)
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
		return fmt.Errorf("invalid HTTP status of response: %d", config.Status)
	}

	// validate template
	if config.IsTemplate && len(config.Body) > 0 {
		if _, err := template.New("body").Parse(config.Body); err != nil {
			return fmt.Errorf("error in body %s", err)
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

	return method, fmt.Errorf("unknown HTTP method: %s", method)
}

// GetBaskets handles HTTP request to get registered baskets
func GetBaskets(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if authorizeRequest(w, r, false, serverConfig) {
		values := r.URL.Query()
		if query := values.Get("q"); len(query) > 0 {
			// find names
			max, skip := getPage(values)
			json, err := json.Marshal(basketsDb.FindNames(query, max, skip))
			writeJSON(w, http.StatusOK, json, err)
		} else {
			// get basket names page
			json, err := json.Marshal(basketsDb.GetNames(getPage(values)))
			writeJSON(w, http.StatusOK, json, err)
		}
	}
}

// GetStats handles HTTP request to get database statistics
func GetStats(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if authorizeRequest(w, r, false, serverConfig) {
		// get database stats
		max := parseInt(r.URL.Query().Get("max"), 1, 100, 5)
		json, err := json.Marshal(basketsDb.GetStats(max))
		writeJSON(w, http.StatusOK, json, err)
	}
}

// GetVersion handles HTTP request to get service version details
func GetVersion(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// get database stats
	json, err := json.Marshal(version)
	writeJSON(w, http.StatusOK, json, err)
}

// GetBasket handles HTTP request to get basket configuration
func GetBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAuthorizedBasket(w, r, ps, serverConfig); basket != nil {
		json, err := json.Marshal(basket.Config())
		writeJSON(w, http.StatusOK, json, err)
	}
}

// CreateBasket handles HTTP request to create a new basket
func CreateBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if !authorizeRequest(w, r, true, serverConfig) {
		return
	}

	name := ps.ByName("basket")
	if name == serviceOldAPIPath || name == serviceAPIPath || name == serviceUIPath {
		http.Error(w, "This basket name conflicts with reserved system path: "+name, http.StatusForbidden)
		return
	}
	if !validBasketName.MatchString(name) {
		http.Error(w, "invalid basket name; the name does not match pattern: "+validBasketName.String(), http.StatusBadRequest)
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
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
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
	if _, basket := getAuthorizedBasket(w, r, ps, serverConfig); basket != nil {
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
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
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
	if name, basket := getAuthorizedBasket(w, r, ps, serverConfig); basket != nil {
		log.Printf("[info] deleting basket: %s", name)

		basketsDb.Delete(name)
		w.WriteHeader(http.StatusNoContent)
	}
}

// GetBasketResponse handles HTTP request to get basket response configuration
func GetBasketResponse(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAuthorizedBasket(w, r, ps, serverConfig); basket != nil {
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
	if _, basket := getAuthorizedBasket(w, r, ps, serverConfig); basket != nil {
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
					http.Error(w, err.Error(), http.StatusUnprocessableEntity)
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
	if _, basket := getAuthorizedBasket(w, r, ps, serverConfig); basket != nil {
		values := r.URL.Query()
		if query := values.Get("q"); len(query) > 0 {
			// find requests
			max, skip := getPage(values)
			json, err := json.Marshal(basket.FindRequests(query, values.Get("in"), max, skip))
			writeJSON(w, http.StatusOK, json, err)
		} else {
			// get requests page
			json, err := json.Marshal(basket.GetRequests(getPage(values)))
			writeJSON(w, http.StatusOK, json, err)
		}
	}
}

// ClearBasket handles HTTP request to delete all requests collected by basket
func ClearBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAuthorizedBasket(w, r, ps, serverConfig); basket != nil {
		basket.Clear()
		w.WriteHeader(http.StatusNoContent)
	}
}

// ForwardToWeb handels HTTP forwarding to /web
func ForwardToWeb(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	http.Redirect(w, r, serverConfig.PathPrefix+"/"+serviceUIPath, http.StatusFound)
}

type TemplateData struct {
	Prefix   string
	Version  *Version
	ThemeCSS template.HTML
	Basket   string
	Data     interface{}
}

// WebIndexPage handles HTTP request to render index page
func WebIndexPage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	indexPageTemplate.Execute(w, TemplateData{Prefix: serverConfig.PathPrefix, Version: version, ThemeCSS: serverConfig.ThemeCSS})
}

// WebBasketPage handles HTTP request to render basket details page
func WebBasketPage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if name := ps.ByName("basket"); validBasketName.MatchString(name) {
		switch name {
		case serviceOldAPIPath:
			// admin page to access all baskets
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			basketsPageTemplate.Execute(w, TemplateData{Prefix: serverConfig.PathPrefix, Version: version, ThemeCSS: serverConfig.ThemeCSS})
		default:
			basketPageTemplate.Execute(w, TemplateData{Prefix: serverConfig.PathPrefix, Version: version, ThemeCSS: serverConfig.ThemeCSS, Basket: name})
		}
	} else {
		http.Error(w, "Basket name does not match pattern: "+validBasketName.String(), http.StatusBadRequest)
	}
}

// AcceptBasketRequests accepts and handles HTTP requests passed to different baskets
func AcceptBasketRequests(w http.ResponseWriter, r *http.Request) {
	name, publicErr, err := getBasketNameOfAcceptedRequest(r, serverConfig.PathPrefix)
	if err != nil {
		log.Printf("[error] %s", err)
		http.Error(w, publicErr, http.StatusBadRequest)
	} else if basket := basketsDb.Get(name); basket != nil {
		request := basket.Add(r)

		// forward request if configured and it's a first forwarding
		config := basket.Config()
		if len(config.ForwardURL) > 0 && r.Header.Get(DoNotForwardHeader) != "1" {
			if config.ProxyResponse {
				forwardAndProxyResponse(w, request, config, name)
				return
			}

			go forwardAndForget(request, config, name)
		}

		writeBasketResponse(w, request, name, basket)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func getBasketNameOfAcceptedRequest(r *http.Request, prefix string) (string, string, error) {
	path := r.URL.Path
	if len(prefix) > 0 {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
		} else {
			publicErr := "incoming request is outside of configured path prefix: " + prefix
			return "", publicErr, fmt.Errorf("%s; request: %s %s", publicErr, r.Method, sanitizeForLog(r.URL.Path))
		}
	}

	name := sanitizeForLog(strings.Split(path, "/")[1])
	if !validBasketName.MatchString(name) {
		publicErr := "invalid basket name; the name does not match pattern: " + validBasketName.String()
		return "", publicErr, fmt.Errorf("%s; request: %s %s", publicErr, r.Method, sanitizeForLog(r.URL.Path))
	}

	return name, "", nil
}

func forwardAndForget(request *RequestData, config BasketConfig, name string) {
	// forward request and discard the response
	response, err := request.Forward(getHTTPClient(config.InsecureTLS), config, name)
	if err != nil {
		log.Printf("[warn] failed to forward request for basket: %s - %s", name, err)
	} else {
		io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()
	}
}

func forwardAndProxyResponse(w http.ResponseWriter, request *RequestData, config BasketConfig, name string) {
	// forward request in a full proxy mode
	response, err := request.Forward(getHTTPClient(config.InsecureTLS), config, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		// headers
		for k, v := range response.Header {
			w.Header()[k] = v
		}

		// status
		w.WriteHeader(response.StatusCode)

		// body
		_, err := io.Copy(w, response.Body)
		if err != nil {
			log.Printf("[warn] failed to proxy response body for basket: %s - %s", name, err)
			io.Copy(ioutil.Discard, response.Body)
		}
		response.Body.Close()
	}
}

func writeBasketResponse(w http.ResponseWriter, request *RequestData, name string, basket Basket) {
	response := basket.GetResponse(request.Method)
	if response == nil {
		response = &defaultResponse
	}

	// headers
	for k, v := range response.Headers {
		w.Header()[k] = v
	}

	// body
	if response.IsTemplate && len(response.Body) > 0 {
		// template
		t, err := template.New(name + "-" + request.Method).Parse(response.Body)
		if err != nil {
			// invalid template
			http.Error(w, "Error in "+err.Error(), http.StatusInternalServerError)
		} else {
			// status
			w.WriteHeader(response.Status)
			// templated body
			t.Execute(w, createTemplateData(request))
		}
	} else {
		// status
		w.WriteHeader(response.Status)
		// plain body
		w.Write([]byte(response.Body))
	}
}

func sanitizeForLog(raw string) string {
	sanitized := strings.ReplaceAll(raw, "\n", "^n")
	sanitized = strings.ReplaceAll(sanitized, "\r", "^r")
	return sanitized
}
