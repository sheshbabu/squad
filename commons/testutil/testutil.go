package testutil

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"squad/commons/sqlite"
	"strconv"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func SetupDB(t *testing.T) {
	t.Helper()

	folder := t.TempDir()
	t.Setenv("DATA_FOLDER", folder)

	db, err := sql.Open("sqlite3", "file:"+filepath.Join(folder, "test.db")+"?_foreign_keys=on&_busy_timeout=5000&_txlock=immediate")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	_, currentFile, _, _ := runtime.Caller(0)
	paths, err := filepath.Glob(filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations", "*.sql"))
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(paths, func(i, j int) bool {
		return getVersion(paths[i]) < getVersion(paths[j])
	})

	for _, path := range paths {
		schema, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(schema)); err != nil {
			t.Fatalf("applying %s: %v", filepath.Base(path), err)
		}
	}

	sqlite.DB = db
}

func getVersion(path string) int {
	version, _ := strconv.Atoi(strings.Split(filepath.Base(path), "_")[0])
	return version
}
