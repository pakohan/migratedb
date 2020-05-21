package main

import (
	"crypto/md5"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

var regexFilename = regexp.MustCompile(`(\d{2})__([a-zA-Z0-9_-]+).sql`)

func main() {
	var dbfile, dir string
	flag.StringVar(&dbfile, "conn", "", "db connection string")
	flag.StringVar(&dir, "dir", "", "migration file directory")
	flag.Parse()

	if dbfile == "" || dir == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	db, err := sql.Open("sqlite3", dbfile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = start(db, dir)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func start(db *sql.DB, dir string) error {
	files, err := getMigrationFiles(dir)
	if err != nil {
		return err
	}

	migrations := make([]migration, len(files))
	for i, file := range files {
		migrations[i], err = initMigration(dir, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func getMigrationFiles(dir string) ([]string, error) {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := []string{}
	skipped := 0
	for _, fi := range fis {
		if fi.IsDir() || !regexFilename.MatchString(fi.Name()) {
			skipped++
			continue
		}

		files = append(files, fi.Name())
	}
	sort.Strings(files)

	log.Printf("found %d migrations, skipped %d elements", len(files), skipped)
	return files, nil
}

func initMigration(dir, file string) (migration, error) {
	parts := regexFilename.FindStringSubmatch(file)
	if len(parts) != 3 {
		return migration{}, fmt.Errorf("expected three parts from regex, got %d (%s)", len(parts), parts)
	}

	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return migration{}, err
	}

	f, err := os.Open(filepath.Join(dir, file))
	if err != nil {
		return migration{}, err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return migration{}, err
	}

	return migration{
		ID:      id,
		Title:   parts[2],
		MD5Sum:  fmt.Sprintf("%x", md5.Sum(b)),
		content: string(b),
	}, nil
}

const (
	createTable = `
CREATE TABLE "migrations" (
  "id"      INTEGER NOT NULL PRIMARY KEY UNIQUE,
  "title"   TEXT NOT NULL UNIQUE,
  "md5_sum" TEXT NOT NULL UNIQUE
)`
)

type migration struct {
	ID      int    `db:"id"`
	Title   string `db:"title"`
	MD5Sum  string `db:"md5_sum"`
	content string
}
