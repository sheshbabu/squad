package sqlite

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"squad/commons/utils"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB = nil

// Workers, the scheduler and handlers write concurrently.
// busy_timeout makes a writer wait for the lock instead of failing, and txlock=immediate takes the write lock at BEGIN so a read-then-write transaction can't deadlock on lock upgrade.
var pragmas = "?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=on&_busy_timeout=5000&_txlock=immediate"

func NewDB() {
	folder := utils.GetDataFolderPath()
	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		panic(fmt.Errorf("error creating data folder: %w", err))
	}

	path := utils.GetDataFolderPath("squad.db")
	slog.Info("connecting to sqlite file", "path", path)

	var err error
	DB, err = sql.Open("sqlite3", "file:"+path+pragmas)
	if err != nil {
		panic(fmt.Errorf("error connecting to sqlite file: %w", err))
	}
}
