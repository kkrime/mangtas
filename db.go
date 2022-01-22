package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// using gorm ORM library

// for db config file
type dbConfig struct {
	Host     string
	Port     string
	Dbname   string
	Username string
	Password string
}

const (
	// db config file location
	LOCAL_DB_CONFIG = "./config/dbConfig.json"

	// db config file error message
	CONFIG_FILE_ERROR = "error with config file %v error %v"
)

var (
	// db
	db *gorm.DB
)

func init() {

	// db logger
	dbLogger := logger.New(
		log.New(),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// read db config file
	dbconfig := readDBConfig()

	var err error

	// attempt to connect to database
	dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=disable TimeZone=UTC",
		dbconfig.Host, dbconfig.Username, dbconfig.Password, dbconfig.Dbname, dbconfig.Port)
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: dbLogger})
	if err != nil {
		// if db does not exist
		if strings.Contains(err.Error(), "SQLSTATE 3D000") {

			// close db connection
			closeDBconnection()

			// connect to db server without specifying a db
			dsn = fmt.Sprintf("host=%v user=%v password=%v port=%v sslmode=disable TimeZone=UTC",
				dbconfig.Host, dbconfig.Username, dbconfig.Password, dbconfig.Port)
			db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
				Logger: dbLogger})
			if err != nil {
				log.Panic("failed to connect database")
			}

			// create db
			stmt := fmt.Sprintf("CREATE DATABASE %s;", dbconfig.Dbname)
			if rs := db.Exec(stmt); rs.Error != nil {
				log.Panic(rs.Error)
			}

			// close db connection
			closeDBconnection()

			// reconnect to newly created db
			dsn = fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=disable TimeZone=UTC",
				dbconfig.Host, dbconfig.Username, dbconfig.Password, dbconfig.Dbname, dbconfig.Port)
			db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
				Logger: dbLogger})
			if err != nil {
				log.Panic("failed to connect database")
			}
		} else {
			log.Panic("failed to connect database")
		}

	}

	// create/sanity check
	err = db.AutoMigrate(&Word{})
	if err != nil {
		log.Panic("migration failed")
	}
}

func readDBConfig() dbConfig {
	var dbconfig dbConfig

	configFile := LOCAL_DB_CONFIG

	config, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Panicf(CONFIG_FILE_ERROR, configFile, err.Error())
	}

	if err = json.Unmarshal(config, &dbconfig); err != nil {
		log.Panicf("Unable to marshal config file %v error %v", configFile, err.Error())
	}

	return dbconfig

}

func closeDBconnection() {
	sql, err := db.DB()
	if err != nil {
		log.Panicf("Unable to close get *sql.DB")
	}
	err = sql.Close()
	if err != nil {
		log.Panicf("Unable to close DB connection")
	}
}

// word is a row in the 'words' db table
type Word struct {
	gorm.Model
	Word  string `gorm:"uniqueIndex"`
	Count int
}

// each word is added in its own thread
func addWordsDB(wordCountMap map[string]int) error {

	// channels to send errors/let us know when threads have finished running
	errorChans := make([]chan struct{}, len(wordCountMap))
	// tx for each word/thread
	txs := make([]*gorm.DB, len(wordCountMap))

	// i to keep track of loop iteration count
	i := 0
	for word, count := range wordCountMap {

		// channel for thread
		errorChans[i] = make(chan struct{})

		// goroutine to connect + add word to db
		go func(txIndex int, word string, count int, errorChan chan<- struct{}) {
			// create new db tx
			txs[txIndex] = db.Begin()
			tx := txs[txIndex]

			var err error
			// on exit log + send errors through error chan and close error chan
			defer func() {
				if err != nil {
					log.Error(err)
					errorChan <- struct{}{}
				}
				close(errorChan)
			}()

			// create empty word record
			wordRecord := Word{Word: word}

			// try to get word if exists, if not create word record in db and lock it
			if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("word", word).
				FirstOrCreate(&wordRecord).Error; err != nil {
				return
			}

			// increment count of word
			wordRecord.Count += count
			// save record to db
			err = tx.Save(wordRecord).Error

		}(i, word, count, errorChans[i])

		i += 1
	}

	// check if any of the threads failed by checking the errorchans
	errorr := false
	for _, errChan := range errorChans {
		for _ = range errChan {
			errorr = true
			// no break because it might cause a goroutine leak and life is too short
			// break
		}
	}

	// talk about the corner case

	// all or nothing commit to db
	if errorr {
		// go func(txs []*gorm.DB) {
		for _, tx := range txs {
			tx.Rollback()
		}
		// }(txs)
		return fmt.Errorf("internal error")
	} else {
		// go func(txs []*gorm.DB) {
		for _, tx := range txs {
			tx.Commit()
		}
		// }(txs)
	}

	return nil
}

func getTopWordsDB() ([]wordOut, error) {

	// NOTE using wordOut struct and not words
	var topWords []wordOut
	err := db.Table("words").
		Order("count desc").
		Limit(10).
		Find(&topWords).Error

	return topWords, err

}
