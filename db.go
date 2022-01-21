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

type DBConfig struct {
	Host     string
	Port     string
	Dbname   string
	Username string
	Password string
}

const (
	LOCAL_DB_CONFIG = "./config/dbConfig.json"

	CONFIG_FILE_ERROR = "error with config file %v error %v"
)

var (
	db *gorm.DB
)

func init() {

	newLogger := logger.New(
		// log.New(os.Stdout, "\r\n", log.LstdFlags),
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
		// Logger: newLogger})
		Logger: newLogger})
	if err != nil {
		// db does not exist
		if strings.Contains(err.Error(), "SQLSTATE 3D000") {

			closeDBconnection()

			// connect to db server without specifying a db
			dsn = fmt.Sprintf("host=%v user=%v password=%v port=%v sslmode=disable TimeZone=UTC",
				dbconfig.Host, dbconfig.Username, dbconfig.Password, dbconfig.Port)
			db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
				Logger: newLogger})
			if err != nil {
				log.Panic("failed to connect database")
			}

			// create db
			stmt := fmt.Sprintf("CREATE DATABASE %s;", dbconfig.Dbname)
			if rs := db.Exec(stmt); rs.Error != nil {
				fmt.Println(rs.Error)
				// return rs.Error
			}

			closeDBconnection()

			// reconnect to db
			dsn = fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=disable TimeZone=UTC",
				dbconfig.Host, dbconfig.Username, dbconfig.Password, dbconfig.Dbname, dbconfig.Port)
			db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
				Logger: newLogger})
			if err != nil {
				log.Panic("failed to connect database")
			}
		} else {
			log.Panic("failed to connect database")
		}

	}

	// sanity check
	err = db.AutoMigrate(&Words{})
	if err != nil {
		log.Panic("migration failed")
	}
}

func readDBConfig() DBConfig {
	var dbconfig DBConfig

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

type Words struct {
	gorm.Model
	Word  string `gorm:"uniqueIndex"`
	Count int
}

func addWordsDB(words []string) error {
	err := db.Transaction(func(tx *gorm.DB) error {

		errorChans := make([]chan error, len(words))

		for i, word := range words {

			errorChans[i] = make(chan error)

			go func(word string, errorChan chan<- error) {
				defer func() {
					close(errorChan)
				}()

				err := db.Transaction(func(tx *gorm.DB) error {

					lowerCaseWord := strings.ToLower(word)

					wordRecord := Words{Word: lowerCaseWord}

					if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
						Where("word", word).
						FirstOrCreate(&wordRecord).Error; err != nil {
						return err
					}

					wordRecord.Count += 1
					tx.Save(wordRecord)
					return nil
				})

				if err != nil {
					errorChan <- err
				}

			}(word, errorChans[i])
		}

		errorr := false
		for _, errChan := range errorChans {
			for err := range errChan {
				errorr = true

				// log error
				log.Error(err)
			}
		}

		if errorr {
			return fmt.Errorf("internal error")
		}

		return nil
	})
	return err

}
