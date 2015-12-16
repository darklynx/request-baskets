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

var basketDb = MakeBasketDb()
var httpClient = new(http.Client)

func writeJson(w http.ResponseWriter, status int, json []byte, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(status)
		w.Write(json)
	}
}

func getIntParam(r *http.Request, name string, defaultValue int) int {
	value := r.URL.Query().Get(name)
	if len(value) > 0 {
		i, err := strconv.Atoi(value)
		if err == nil {
			return i
		}
	}
	return defaultValue
}

func getAndAuthBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) (string, *Basket) {
	name := ps.ByName("basket")
	basket := basketDb.Get(name)
	if basket != nil {
		// maybe custom header, e.g. basket_key, basket_token
		token := r.Header.Get("Authorization")
		if token == basket.Token || token == serverConfig.MasterToken {
			return name, basket
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}

	return "", nil
}

func parseConfig(body []byte, config *Config) error {
	// parse request
	if err := json.Unmarshal(body, config); err != nil {
		return err
	}

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

func GetBaskets(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	json, err := basketDb.ToJson(
		getIntParam(r, "max", serverConfig.PageSize),
		getIntParam(r, "skip", 0))
	writeJson(w, http.StatusOK, json, err)
}

func GetBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAndAuthBasket(w, r, ps); basket != nil {
		json, err := basket.ToJson()
		writeJson(w, http.StatusOK, json, err)
	}
}

func CreateBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	name := ps.ByName("basket")
	if name == BASKETS_ROOT || name == WEB_ROOT {
		http.Error(w, "You cannot use system path as basket name: "+name, http.StatusForbidden)
		return
	}
	if !validBasketName.MatchString(name) {
		http.Error(w, "Invalid basket name: "+name+", valid name pattern: "+validBasketName.String(), http.StatusBadRequest)
		return
	}

	log.Printf("Creating basket: %s", name)

	// read config (max 2 kB)
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 2048))
	r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// default config
	config := Config{ForwardUrl: "", Capacity: serverConfig.InitCapacity}
	if len(body) > 0 {
		if err := parseConfig(body, &config); err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
	}

	basket, err := basketDb.Create(name, config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
	} else {
		json, err := basket.ToAuthJson()
		writeJson(w, http.StatusCreated, json, err)
	}
}

func UpdateBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAndAuthBasket(w, r, ps); basket != nil {
		// read config (max 2 kB)
		body, err := ioutil.ReadAll(io.LimitReader(r.Body, 2048))
		r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else if len(body) > 0 {
			config := basket.Config
			if err := parseConfig(body, &config); err != nil {
				http.Error(w, err.Error(), 422)
				return
			}

			basket.Config = config
			basket.Requests.UpdateCapacity(config.Capacity)

			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusNotModified)
		}
	}
}

func DeleteBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if name, basket := getAndAuthBasket(w, r, ps); basket != nil {
		log.Printf("Deleting basket: %s", name)

		basketDb.Delete(name)
		w.WriteHeader(http.StatusNoContent)
	}
}

func GetBasketRequests(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAndAuthBasket(w, r, ps); basket != nil {
		json, err := basket.Requests.ToJson(
			getIntParam(r, "max", serverConfig.PageSize),
			getIntParam(r, "skip", 0))
		writeJson(w, http.StatusOK, json, err)
	}
}

func ClearBasket(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if _, basket := getAndAuthBasket(w, r, ps); basket != nil {
		basket.Requests.Clear()
		w.WriteHeader(http.StatusNoContent)
	}
}

func ForwardToWeb(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	http.Redirect(w, r, "/web", 302)
}

func WebIndexPage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	indexPage.Execute(w, "")
}
func WebBasketPage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	name := ps.ByName("basket")
	basketPage.Execute(w, name)
}

func AcceptBasketRequests(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	name := parts[1]
	basket := basketDb.Get(name)
	if basket != nil {
		request := basket.Requests.Add(r)
		w.WriteHeader(http.StatusOK)

		if len(basket.Config.ForwardUrl) > 0 {
			go request.Forward(httpClient, basket.Config.ForwardUrl)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
