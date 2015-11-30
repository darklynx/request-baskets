package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
)

type Config struct {
	ForwardUrl string `json:"forward_url"`
	Capacity   int    `json:"capacity"`
}

type Auth struct {
	Token string `json:"token"`
}

type Basket struct {
	Token    string
	Config   Config
	Requests *RequestDb
}

type Baskets struct {
	Names   []string `json:"names"`
	Count   int      `json:"count"`
	HasMore bool     `json:"has_more"`
}

type BasketDb struct {
	sync.RWMutex
	baskets map[string]*Basket
	names   []string
}

var tokenLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func GenerateToken(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = tokenLetters[rand.Intn(len(tokenLetters))]
	}
	return string(b)
}

func (b *Basket) ToJson() ([]byte, error) {
	return json.Marshal(b.Config)
}

func (b *Basket) ToAuthJson() ([]byte, error) {
	auth := Auth{Token: b.Token}
	return json.Marshal(auth)
}

func (db *BasketDb) Create(name string, config Config) (*Basket, error) {
	db.Lock()
	defer db.Unlock()

	_, exists := db.baskets[name]
	if exists {
		return nil, fmt.Errorf("Basket with name '%s' already exists", name)
	}

	basket := new(Basket)
	basket.Token = GenerateToken(20)
	basket.Config = config
	basket.Requests = MakeRequestDb(config.Capacity)

	db.baskets[name] = basket
	db.names = append(db.names, name)
	// Uncomment if sorting is expected
	// sort.Strings(db.names)

	return basket, nil
}

func (db *BasketDb) Get(name string) *Basket {
	basket, _ := db.baskets[name]
	return basket
}

func (db *BasketDb) Delete(name string) {
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

func (db *BasketDb) ToJson(max int, skip int) ([]byte, error) {
	db.RLock()
	defer db.RUnlock()

	size := len(db.names)
	last := skip + max

	baskets := Baskets{
		Count:   size,
		HasMore: last < size}

	if skip < size {
		if last > size {
			last = size
		}

		baskets.Names = db.names[skip:last]
	}

	return json.Marshal(baskets)
}

func MakeBasketDb() *BasketDb {
	return &BasketDb{baskets: make(map[string]*Basket), names: make([]string, 0)}
}
