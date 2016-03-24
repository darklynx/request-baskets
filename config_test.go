package main

import (
	"testing"
)

func TestCreateDefaultConfig(t *testing.T) {
	config := CreateConfig()
	if config == nil {
		t.Fatalf("default configuration is expected")
	}

	if config.DbType != DEFAULT_DB_TYPE {
		t.Fatalf("wrong db type, expected: %v, but found: %v", DEFAULT_DB_TYPE, config.DbType)
	}
	if config.ServerPort != DEFAULT_PORT {
		t.Fatalf("wrong server port, expected: %v, but found: %v", DEFAULT_PORT, config.ServerPort)
	}
	if config.InitCapacity != INIT_BASKET_CAPACITY {
		t.Fatalf("wrong initial capacity, expected: %v, but found: %v", INIT_BASKET_CAPACITY, config.InitCapacity)
	}
	if config.MaxCapacity != MAX_BASKET_CAPACITY {
		t.Fatalf("wrong max capacity, expected: %v, but found: %v", MAX_BASKET_CAPACITY, config.MaxCapacity)
	}
	if config.PageSize != DEFAULT_PAGE_SIZE {
		t.Fatalf("wrong page size, expected: %v, but found: %v", DEFAULT_PAGE_SIZE, config.PageSize)
	}
	if config.DbFile != "./baskets.db" {
		t.Fatalf("wrong page size, expected: ./baskets.db, but found: %v", config.DbFile)
	}

	if len(config.MasterToken) == 0 {
		t.Fatalf("expected randomly generated master token, but was empty")
	}
}
