package database

import (
	"fmt"
	"log"

	sqle "github.com/dolthub/go-mysql-server"
	"github.com/dolthub/go-mysql-server/memory"
	"github.com/dolthub/go-mysql-server/server"
	"github.com/dolthub/go-mysql-server/sql"
	_ "github.com/go-sql-driver/mysql"
)

func SetupLocalDB(address string, port string, dbName string) *DBConnection {
	// Start local mysql engine
	log.Println("starting local mysql db...")
	ctx := sql.NewEmptyContext()
	engine := sqle.NewDefault(
		memory.NewDBProvider(
			createLocalDatabase(ctx, dbName),
		))

	dbConfig := server.Config{
		Protocol: "tcp",
		Address:  fmt.Sprintf("%s:%s", address, port),
	}
	s, err := server.NewDefaultServer(dbConfig, engine)
	if err != nil {
		panic(err)
	}
	go func() {
		if err = s.Start(); err != nil {
			panic(err)
		}
	}()

	return &DBConnection{}
}

func createLocalDatabase(ctx *sql.Context, dbName string) *memory.Database {
	db := memory.NewDatabase(dbName)
	return db
}
