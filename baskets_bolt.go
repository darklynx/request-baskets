package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
)

var (
	KEY_TOKEN       = []byte("token")
	KEY_FORWARD_URL = []byte("url")
	KEY_CAPACITY    = []byte("capacity")
	KEY_TOTAL_COUNT = []byte("total")
	KEY_COUNT       = []byte("count")
	KEY_REQUESTS    = []byte("requests")
)

/// Basket interface ///

type boltBasket struct {
	db   *bolt.DB
	name string
}

func (basket *boltBasket) Config() BasketConfig {
	config := BasketConfig{}

	basket.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(basket.name))
		if b != nil {
			url := b.Get(KEY_FORWARD_URL)
			cap := b.Get(KEY_CAPACITY)

			config.ForwardUrl = string(url[:])
			config.Capacity, _ = strconv.Atoi(string(cap[:]))
		} else {
			log.Printf("Failed to locate bucket: %s", basket.name)
		}

		return nil
	})

	return config
}

/// BasketsDatabase interface ///

type boltDatabase struct {
	db *bolt.DB
}

func (bdb *boltDatabase) Create(name string, config BasketConfig) (BasketAuth, error) {
	auth := BasketAuth{}
	token, err := GenerateToken()
	if err != nil {
		return auth, fmt.Errorf("Failed to generate token: %s", err)
	}

	err = bdb.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte(name))
		if err != nil {
			return fmt.Errorf("Failed to create bucket: %s - %s", name, err)
		}

		// initialize basket bucket (assume no issues arised)
		b.Put(KEY_TOKEN, []byte(token))
		b.Put(KEY_FORWARD_URL, []byte(config.ForwardUrl))
		b.Put(KEY_CAPACITY, []byte(strconv.Itoa(config.Capacity)))
		b.Put(KEY_TOTAL_COUNT, []byte("0"))
		b.Put(KEY_COUNT, []byte("0"))
		b.CreateBucket(KEY_REQUESTS)

		return err
	})

	if err != nil {
		return auth, err
	}

	auth.Token = token

	return auth, nil
}

func (bdb *boltDatabase) Get(name string) *boltBasket {
	err := bdb.db.View(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte(name)) != nil {
			return nil
		} else {
			return fmt.Errorf("No basket found: %s", name)
		}
	})

	if err != nil {
		log.Print(err)
		return nil
	} else {
		return &boltBasket{bdb.db, name}
	}
}

func (bdb *boltDatabase) Delete(name string) {
	err := bdb.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(name))
	})

	if err != nil {
		log.Print(err)
	}
}

func (bdb *boltDatabase) Size() int {
	// TODO : introduce bucket with statistics (e.g. "/stats", or ".stats"), see https://github.com/boltdb/bolt/issues/276
	size := 0

	bdb.db.View(func(tx *bolt.Tx) error {
		cur := tx.Cursor()
		for key, _ := cur.First(); key != nil; key, _ = cur.Next() {
			size++
		}
		return nil
	})

	return size
}

func (bdb *boltDatabase) GetNames(max int, skip int) BasketNamesPage {
	last := skip + max
	page := BasketNamesPage{make([]string, 0, max), 0, false}

	bdb.db.View(func(tx *bolt.Tx) error {
		cur := tx.Cursor()
		for key, _ := cur.First(); key != nil; key, _ = cur.Next() {
			if page.Count >= skip && page.Count < last {
				page.Names[len(page.Names)] = string(key[:])
			} else if page.Count >= last {
				page.HasMore = true
			}
			page.Count++
		}
		return nil
	})

	return page
}

func (bdb *boltDatabase) Release() {
	log.Printf("Releasing bolt database resources")
	err := bdb.db.Close()
	if err != nil {
		log.Print(err)
	}
}

// NewBoltDatabase creates an instance of Baskets Database backed with Bolt DB
func NewBoltDatabase(file string) *boltDatabase {
	log.Printf("Bolt database location: %s", file)
	db, err := bolt.Open(file, 0600, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		log.Fatal(err)
		return nil
	}

	return &boltDatabase{db}
}
