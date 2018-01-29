package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

// DbTypeMemory defines name of in-memory database storage
const DbTypeMemory = "mem"

/// Basket interface ///

type memoryBasket struct {
	sync.RWMutex
	token      string
	config     BasketConfig
	requests   []*RequestData
	totalCount int
	responses  map[string]*ResponseConfig
}

func (basket *memoryBasket) applyLimit() {
	// Keep requests up to specified capacity
	if len(basket.requests) > basket.config.Capacity {
		basket.requests = basket.requests[:basket.config.Capacity]
	}
}

func (basket *memoryBasket) Config() BasketConfig {
	return basket.config
}

func (basket *memoryBasket) Update(config BasketConfig) {
	basket.Lock()
	defer basket.Unlock()

	basket.config = config
	basket.applyLimit()
}

func (basket *memoryBasket) Authorize(token string) bool {
	return token == basket.token
}

func (basket *memoryBasket) GetResponse(method string) *ResponseConfig {
	basket.Lock()
	defer basket.Unlock()

	if response, exists := basket.responses[method]; exists {
		return response
	}

	return nil
}

func (basket *memoryBasket) SetResponse(method string, response ResponseConfig) {
	basket.Lock()
	defer basket.Unlock()

	basket.responses[method] = &response
}

func (basket *memoryBasket) Add(req *http.Request) *RequestData {
	basket.Lock()
	defer basket.Unlock()

	data := ToRequestData(req)
	// insert in front of collection
	basket.requests = append([]*RequestData{data}, basket.requests...)

	// keep total number of all collected requests
	basket.totalCount++
	// apply limits according to basket capacity
	basket.applyLimit()

	return data
}

func (basket *memoryBasket) Clear() {
	basket.Lock()
	defer basket.Unlock()

	// reset collected requests and total counter
	basket.requests = make([]*RequestData, 0, basket.config.Capacity)
	// basket.totalCount = 0 // reset total stats
}

func (basket *memoryBasket) Size() int {
	return len(basket.requests)
}

func (basket *memoryBasket) GetRequests(max int, skip int) RequestsPage {
	basket.RLock()
	defer basket.RUnlock()

	size := basket.Size()
	last := skip + max

	requestsPage := RequestsPage{
		Count:      size,
		TotalCount: basket.totalCount,
		HasMore:    last < size}

	if skip < size {
		if last > size {
			last = size
		}
		requestsPage.Requests = basket.requests[skip:last]
	}

	return requestsPage
}

func (basket *memoryBasket) FindRequests(query string, in string, max int, skip int) RequestsQueryPage {
	basket.RLock()
	defer basket.RUnlock()

	result := make([]*RequestData, 0, max)
	skipped := 0

	for index, request := range basket.requests {
		// filter
		if request.Matches(query, in) {
			if skipped < skip {
				skipped++
			} else {
				result = append(result, request)
			}
		}

		// early exit
		if len(result) == max {
			return RequestsQueryPage{Requests: result, HasMore: index < len(basket.requests)-1}
		}
	}

	// whole basket is scanned through
	return RequestsQueryPage{Requests: result, HasMore: false}
}

/// BasketsDatabase interface ///

type memoryDatabase struct {
	sync.RWMutex
	baskets map[string]*memoryBasket
	names   []string
}

func (db *memoryDatabase) Create(name string, config BasketConfig) (BasketAuth, error) {
	auth := BasketAuth{}
	token, err := GenerateToken()
	if err != nil {
		return auth, fmt.Errorf("Failed to generate token: %s", err)
	}

	db.Lock()
	defer db.Unlock()

	_, exists := db.baskets[name]
	if exists {
		return auth, fmt.Errorf("Basket with name '%s' already exists", name)
	}

	basket := new(memoryBasket)
	basket.token = token
	basket.config = config
	basket.requests = make([]*RequestData, 0, config.Capacity)
	basket.totalCount = 0
	basket.responses = make(map[string]*ResponseConfig)

	db.baskets[name] = basket
	db.names = append(db.names, name)
	// Uncomment if sorting is expected
	// sort.Strings(db.names)

	auth.Token = token

	return auth, nil
}

func (db *memoryDatabase) Get(name string) Basket {
	if basket, exists := db.baskets[name]; exists {
		return basket
	}

	log.Printf("[warn] no basket found: %s", name)
	return nil
}

func (db *memoryDatabase) Delete(name string) {
	db.Lock()
	defer db.Unlock()

	delete(db.baskets, name)
	for i, v := range db.names {
		if v == name {
			db.names = append(db.names[:i], db.names[i+1:]...)
			break
		}
	}
}

func (db *memoryDatabase) Size() int {
	return len(db.names)
}

func (db *memoryDatabase) GetNames(max int, skip int) BasketNamesPage {
	db.RLock()
	defer db.RUnlock()

	size := len(db.names)
	last := skip + max

	namesPage := BasketNamesPage{
		Count:   size,
		HasMore: last < size}

	if skip < size {
		if last > size {
			last = size
		}

		namesPage.Names = db.names[skip:last]
	}

	return namesPage
}

func (db *memoryDatabase) FindNames(query string, max int, skip int) BasketNamesQueryPage {
	db.RLock()
	defer db.RUnlock()

	result := make([]string, 0, max)
	skipped := 0

	for index, name := range db.names {
		// filter
		if strings.Contains(name, query) {
			if skipped < skip {
				skipped++
			} else {
				result = append(result, name)
			}
		}

		// early exit
		if len(result) == max {
			return BasketNamesQueryPage{Names: result, HasMore: index < len(db.names)-1}
		}
	}

	// whole database is scanned through
	return BasketNamesQueryPage{Names: result, HasMore: false}
}

func (db *memoryDatabase) Release() {
	log.Print("[info] releasing in-memory database resources")
}

// NewMemoryDatabase creates an instance of in-memory Baskets Database
func NewMemoryDatabase() BasketsDatabase {
	log.Print("[info] using in-memory database to store baskets")
	return &memoryDatabase{baskets: make(map[string]*memoryBasket), names: make([]string, 0)}
}
