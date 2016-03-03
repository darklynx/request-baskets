package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestBasketsDatabase_Create(t *testing.T) {
	db := NewMemoryDatabase()
	defer db.Release()

	name := "test1"
	auth, err := db.Create(name, BasketConfig{Capacity: 20})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(auth.Token) == 0 {
		t.Fatalf("basket token is expected")
	}
	if len(auth.Token) < 30 {
		t.Fatalf("insecure token is generated: %v", auth.Token)
	}
}

func TestBasketsDatabase_Create_NameConflict(t *testing.T) {
	db := NewMemoryDatabase()
	defer db.Release()

	name := "test2"
	db.Create(name, BasketConfig{Capacity: 20})
	auth, err := db.Create(name, BasketConfig{Capacity: 20})
	if err == nil {
		t.Fatalf("error is expected")
	}
	if !strings.Contains(err.Error(), "'"+name+"'") {
		t.Fatalf("error is not detailed enough: %v", err)
	}
	if len(auth.Token) > 0 {
		t.Fatalf("token is not expected, but was: %v", auth.Token)
	}
}

func TestBasketsDatabase_Get(t *testing.T) {
	db := NewMemoryDatabase()
	defer db.Release()

	name := "test3"
	auth, err := db.Create(name, BasketConfig{Capacity: 16})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	basket := db.Get(name)
	if basket == nil {
		t.Fatalf("basket with name: %v is expected", name)
	}

	if !basket.Authorize(auth.Token) {
		t.Fatalf("wrong basket, authorization with token: %v has failed", auth.Token)
	}

	if basket.Config().Capacity != 16 {
		t.Fatalf("wrong basket capacity: %v", basket.Config().Capacity)
	}
}

func TestBasketsDatabase_Get_NotFound(t *testing.T) {
	db := NewMemoryDatabase()
	defer db.Release()

	name := "test4"
	basket := db.Get(name)
	if basket != nil {
		t.Fatalf("basket with name: %v is not expected", name)
	}
}

func TestBasketsDatabase_Delete(t *testing.T) {
	db := NewMemoryDatabase()
	defer db.Release()

	name := "test5"
	db.Create(name, BasketConfig{Capacity: 10})
	if db.Get(name) == nil {
		t.Fatalf("basket with name: %v is expected", name)
	}

	db.Delete(name)

	if db.Get(name) != nil {
		t.Fatalf("basket with name: %v is not expected", name)
	}
}

func TestBasketsDatabase_Delete_Multi(t *testing.T) {
	db := NewMemoryDatabase()
	defer db.Release()

	name := "test6"
	config := BasketConfig{Capacity: 10}
	for i := 0; i < 10; i++ {
		db.Create(fmt.Sprintf("test%d", i), config)
	}

	if db.Get(name) == nil {
		t.Fatalf("basket with name: %v is expected", name)
	}
	if db.Size() != 10 {
		t.Fatalf("wrong database size: %d", db.Size())
	}

	db.Delete(name)

	if db.Get(name) != nil {
		t.Fatalf("basket with name: %v is not expected", name)
	}
	if db.Size() != 9 {
		t.Fatalf("wrong database size: %d", db.Size())
	}
}
