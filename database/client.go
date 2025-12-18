package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type DBConnection struct {
	Database  *sql.DB
	Connected bool
}

// Connect initializes the database connection and waits for it to become available or times out after a specified duration
func (dbConn *DBConnection) Connect(username, password, host, port, dbName string, timeout time.Duration) error {
	if dbName == "" {
		log.Println("No Database specified, skipping..")
		dbConn.Connected = false
		return nil
	}

	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", username, password, host, port, dbName)
	var err error
	dbConn.Database, err = sql.Open("mysql", connectionString)
	if err != nil {
		return err
	}

	startTime := time.Now()
	for {
		err = dbConn.Database.Ping()
		if err == nil {
			log.Printf("Connected to the database: [%s]", dbName)
			dbConn.Connected = true
			return nil
		}

		if time.Since(startTime) >= timeout {
			return fmt.Errorf("connection to the database: [%s] timed out after %v", dbName, timeout)
		}

		time.Sleep(1 * time.Second)
	}
}

// ExecuteQuery executes a single SQL query
func (dbConn *DBConnection) ExecuteQuery(query string, args ...interface{}) (sql.Result, error) {
	result, err := dbConn.Database.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// QueryRow executes a query that is expected to return a single row
func (dbConn *DBConnection) QueryRow(query string, args ...interface{}) *sql.Row {
	row := dbConn.Database.QueryRow(query, args...)
	return row
}

// QueryRows executes a query that is expected to return multiple rows
func (dbConn *DBConnection) QueryRows(query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := dbConn.Database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

