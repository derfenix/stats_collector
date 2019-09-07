package collector

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"
)

const (
	tableName = "stats_counters"

	// databaseWaitIntervalSec interval in seconds between database availability checks
	databaseWaitIntervalSec = 10

	// maxDbWriteRetries count of attempts to write data into database before failing
	maxDbWriteRetries = 5
)

var (
	// createTableSQL SQL code will be used for create table query
	createTableSQL = `
CREATE TABLE IF NOT EXISTS ` + tableName + `
(
	id int NOT NULL,
	label VARCHAR(90) NOT NULL,
	counter bigint default 0,
	PRIMARY KEY(id, label)
)
ENGINE MyISAM;`
)

// Storage is the main struct for storing statistics
//
// Uses in-memory cache for incoming data and flush it to the db by ticker signal in `Syncer` method
type Storage struct {
	db        *sql.DB
	insertStm *sql.Stmt
	writerCh  chan *Entity

	mu    sync.RWMutex
	cache map[uint64]map[string]*cacheItem
}

// NewStorage creates new storage
//
// Also prepare table and query
func NewStorage(db *sql.DB) *Storage {
	return &Storage{
		db:        db,
		insertStm: prepareDB(db),

		cache:    make(map[uint64]map[string]*cacheItem, 50),
		writerCh: make(chan *Entity, 500),
	}
}

// prepareDB waits for database availability, create table if not exists and prepared query for writing stats into db
func prepareDB(db *sql.DB) *sql.Stmt {
	log.Print("waiting for database")
	for {
		_, err := db.Exec("SELECT 1")
		if err != nil {
			log.Printf("wait %d sec for database: %s", databaseWaitIntervalSec, err.Error())
			time.Sleep(time.Second * databaseWaitIntervalSec)
		} else {
			log.Print("database online!")
			break
		}
	}
	// Create table
	_, err := db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("failed to create table: %s", err.Error())
	}

	// Prepare sql statement
	stm, err := db.Prepare("INSERT INTO " + tableName + "(id, label, counter) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE counter=counter+?")
	if err != nil {
		log.Fatalf("failed to prepare statement: %s", err.Error())
	}

	return stm
}

// Syncer flush collected statistics in database by Ticker's signal
//
// Creates additional goroutine, which waits data from `Storage.writeCh` and write received data into database
func (s *Storage) Syncer(ctx context.Context, wg *sync.WaitGroup, period time.Duration) {
	defer wg.Done()

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case message, ok := <-s.writerCh:
				if !ok {
					log.Fatal("write channel closed, shutting down")
					return
				}
				_, err := s.insertStm.ExecContext(
					ctx,

					message.ID,
					message.Label,
					message.Count,
					message.Count,
				)
				if err != nil {
					if message.attempt < maxDbWriteRetries {
						message.attempt += 1
						s.writerCh <- message
					} else {
						log.Printf("max attermpts for storing %s counter exceeded", message.Label)
					}
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Print("flushing")
			s.sync(ctx)
		}
	}
}

// sync flush each cache cell and sends values from them into `Storage.writeCh`
func (s *Storage) sync(ctx context.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for id, mm := range s.cache {
		for label, cache := range mm {
			s.writerCh <- &Entity{
				ID:      id,
				Label:   label,
				Count:   cache.flush(),
				attempt: 0,
			}
		}
	}
}

// Add increases counter for specified id and label for provided value
func (s *Storage) Add(id uint64, label string, v uint64) {
	s.ensureStruct(id, label)
	s.cache[id][label].add(v)
}

func (s *Storage) ensureStruct(id uint64, label string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.cache[id]; !ok {
		s.cache[id] = make(map[string]*cacheItem, 10)
	}
	if _, ok := s.cache[id][label]; !ok {
		s.cache[id][label] = new(cacheItem)
	}
}

// cacheItem in-memory cache cell
type cacheItem struct {
	mu      sync.Mutex
	counter uint64
}

func (c *cacheItem) add(v uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.counter += v
}

// flush resets cache cell to zero and returns previous value
func (c *cacheItem) flush() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	v := c.counter
	c.counter = 0
	return v
}
