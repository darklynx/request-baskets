package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
)

const DB_TYPE_BOLT = "bolt"

var (
	KEY_TOKEN       = []byte("token")
	KEY_FORWARD_URL = []byte("url")
	KEY_CAPACITY    = []byte("capacity")
	KEY_TOTAL_COUNT = []byte("total")
	KEY_COUNT       = []byte("count")
	KEY_REQUESTS    = []byte("requests")
)

func itob(i int) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(i))
	return b
}

func btoi(b []byte) int {
	return int(binary.BigEndian.Uint32(b))
}

/// Basket interface ///

type boltBasket struct {
	db   *bolt.DB
	name string
}

func (basket *boltBasket) update(fn func(*bolt.Bucket) error) error {
	err := basket.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(basket.name))
		if b != nil {
			return fn(b)
		} else {
			return fmt.Errorf("failed to locate bucket by name")
		}
	})

	if err != nil {
		log.Printf("[error] %s; basket: %s", err, basket.name)
	}

	return err
}

func (basket *boltBasket) view(fn func(*bolt.Bucket) error) error {
	err := basket.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(basket.name))
		if b != nil {
			return fn(b)
		} else {
			return fmt.Errorf("failed to locate bucket by name")
		}
	})

	if err != nil {
		log.Printf("[error] %s; basket: %s", err, basket.name)
	}

	return err
}

func (basket *boltBasket) Config() BasketConfig {
	config := BasketConfig{}

	basket.view(func(b *bolt.Bucket) error {
		config.ForwardUrl = string(b.Get(KEY_FORWARD_URL))
		config.Capacity = btoi(b.Get(KEY_CAPACITY))

		return nil
	})

	return config
}

func (basket *boltBasket) Update(config BasketConfig) {
	basket.update(func(b *bolt.Bucket) error {
		oldCap := btoi(b.Get(KEY_CAPACITY))
		curCount := btoi(b.Get(KEY_COUNT))

		b.Put(KEY_FORWARD_URL, []byte(config.ForwardUrl))
		b.Put(KEY_CAPACITY, itob(config.Capacity))

		if oldCap != config.Capacity && curCount > config.Capacity {
			// remove overflow requests
			remCount := curCount - config.Capacity

			reqsCur := b.Bucket(KEY_REQUESTS).Cursor()
			reqsCur.First()
			for i := 0; i < remCount; i++ {
				reqsCur.Delete()
				reqsCur.Next()
			}

			// update count
			b.Put(KEY_COUNT, itob(config.Capacity))
		}

		return nil
	})
}

func (basket *boltBasket) Authorize(token string) bool {
	result := false

	basket.view(func(b *bolt.Bucket) error {
		result = string(b.Get(KEY_TOKEN)) == token
		return nil
	})

	return result
}

func (basket *boltBasket) Add(req *http.Request) *RequestData {
	data := ToRequestData(req)

	basket.update(func(b *bolt.Bucket) error {
		reqs := b.Bucket(KEY_REQUESTS)

		dataj, err := json.Marshal(data)
		if err != nil {
			return err
		}

		key, _ := reqs.NextSequence()
		err = reqs.Put(itob(int(key)), dataj)
		if err != nil {
			return err
		}

		// update counters
		cap := btoi(b.Get(KEY_CAPACITY))
		count := btoi(b.Get(KEY_COUNT))
		total := btoi(b.Get(KEY_TOTAL_COUNT))

		// total count
		total++
		b.Put(KEY_TOTAL_COUNT, itob(total))

		// current count (may not exceed capacity)
		if count < cap {
			count++
			b.Put(KEY_COUNT, itob(count))
		} else {
			// do not increase counter, just remove 1 entry
			cur := reqs.Cursor()
			cur.First()
			cur.Delete()

			if count > cap {
				// should not happen
				log.Printf("[warn] number of requests: %d exceeds capacity: %d; basket: %s", count, cap, basket.name)
			}
		}

		return nil
	})

	return data
}

func (basket *boltBasket) Clear() {
	basket.update(func(b *bolt.Bucket) error {
		err := b.DeleteBucket(KEY_REQUESTS)
		if err != nil {
			return err
		}

		b.Put(KEY_TOTAL_COUNT, itob(0))
		b.Put(KEY_COUNT, itob(0))
		b.CreateBucket(KEY_REQUESTS)

		return nil
	})
}

func (basket *boltBasket) Size() int {
	result := -1

	basket.view(func(b *bolt.Bucket) error {
		result = btoi(b.Get(KEY_COUNT))

		return nil
	})

	return result
}

func (basket *boltBasket) GetRequests(max int, skip int) RequestsPage {
	last := skip + max
	page := RequestsPage{make([]*RequestData, 0, max), 0, 0, false}

	basket.view(func(b *bolt.Bucket) error {
		page.TotalCount = btoi(b.Get(KEY_TOTAL_COUNT))
		page.Count = btoi(b.Get(KEY_COUNT))

		cur := b.Bucket(KEY_REQUESTS).Cursor()
		index := 0
		for key, val := cur.Last(); key != nil; key, val = cur.Prev() {
			if index >= skip && index < last {
				request := new(RequestData)
				if err := json.Unmarshal(val, request); err != nil {
					return err
				}
				page.Requests = append(page.Requests, request)
			} else if index >= last {
				page.HasMore = true
				break
			}
			index++
		}

		return nil
	})

	return page
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
			return fmt.Errorf("Failed to create basket: %s - %s", name, err)
		}

		// initialize basket bucket (assume no issues arised)
		b.Put(KEY_TOKEN, []byte(token))
		b.Put(KEY_FORWARD_URL, []byte(config.ForwardUrl))
		b.Put(KEY_CAPACITY, itob(config.Capacity))
		b.Put(KEY_TOTAL_COUNT, itob(0))
		b.Put(KEY_COUNT, itob(0))
		b.CreateBucket(KEY_REQUESTS)

		return nil
	})

	if err != nil {
		return auth, err
	}

	auth.Token = token

	return auth, nil
}

func (bdb *boltDatabase) Get(name string) Basket {
	err := bdb.db.View(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte(name)) != nil {
			return nil
		} else {
			return fmt.Errorf("[warn] no basket found: %s", name)
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
				page.Names = append(page.Names, string(key))
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
	log.Printf("[info] closing bolt database")
	err := bdb.db.Close()
	if err != nil {
		log.Print(err)
	}
}

// NewBoltDatabase creates an instance of Baskets Database backed with Bolt DB
func NewBoltDatabase(file string) BasketsDatabase {
	log.Print("[info] using bolt DB to store baskets")
	log.Printf("[info] bolt database location: %s", file)
	db, err := bolt.Open(file, 0600, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		log.Fatal(err)
		return nil
	}

	return &boltDatabase{db}
}
