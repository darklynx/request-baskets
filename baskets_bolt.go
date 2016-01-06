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
	KEY_REQUESTS    = []byte("requests")
)

/// Basket interface ///

type boltBasket struct {
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
		b.CreateBucketIfNotExists(KEY_REQUESTS)

		return err
	})

	if err != nil {
		return auth, err
	}

	auth.Token = token

	return auth, nil
}

func (bdb *boltDatabase) Release() {
	log.Printf("Releasing bolt database resources")
	err := bdb.db.Close()
	if err != nil {
		log.Fatal(err)
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
