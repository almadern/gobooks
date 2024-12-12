package db

import (
	"gobook/arguments"
	"log"
	"sync"
	"sync/atomic"
)

var (
	DBinitialized uint32
	DBInitMux     sync.Mutex
)

func InitDB() error {
	if atomic.LoadUint32(&DBinitialized) == 1 {
		return nil
	}
	DBInitMux.Lock()
	log.Println("Start working with DB")
	switch arguments.Config.DB {
	case "postgres":
		PostgresInitDB()
	case "sqlite":
		SQLiteInitDB()
	}
	defer DBInitMux.Unlock()

	if atomic.LoadUint32(&DBinitialized) == 1 {
		return nil
	}

	atomic.StoreUint32(&DBinitialized, 1)
	return nil

}
