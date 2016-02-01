package cross

import (
	"database/sql"
	"strings"
	// "log"
	// "os"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/gorp.v1"
)

type Database struct {
	*gorp.DbMap
}

func NewDatabase(uri string) *Database {
	db, err := sql.Open("mysql", uri)
	if err != nil {
		panic(err)
	}

	dbMap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}
	dbMap.AddTableWithName(LogEntry{}, "logEntries").SetKeys(true, "ID")
	dbMap.AddTableWithName(entryContent{}, "submissionContents").SetKeys(true, "ID")
	// dbMap.TraceOn("SQL: ", log.New(os.Stdout, "", 5))
	return &Database{dbMap}
}

func (db *Database) getContent(hash []byte) (*entryContent, error) {
	var entry entryContent
	err := db.SelectOne(
		&entry,
		"SELECT * FROM submissionConents WHERE hash = ?",
		hash,
	)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (db *Database) GetEntry(hash []byte) (*LogEntry, error) {
	obj, err := db.Get(LogEntry{}, hash)
	if err != nil {
		return nil, err
	}
	entry := obj.(*LogEntry)
	content, err := db.getContent(entry.SubmissionHash)
	if err != nil {
		return nil, err
	}
	entry.Submission = content.Content
	return entry, nil
}

func (db *Database) addEntryContent(c *entryContent) error {
	err := db.Insert(c)
	if err != nil && !strings.HasPrefix(err.Error(), "Error 1062: Duplicate entry") {
		return err
	}
	return nil
}

func (db *Database) AddEntry(e *LogEntry) error {
	err := db.Insert(e)
	if err != nil {
		return err
	}
	content := &entryContent{
		Hash:    e.SubmissionHash,
		Content: e.Submission,
	}
	return db.addEntryContent(content)
}

func (db *Database) getCurrentLogIndex(logID []byte) (int64, error) {
	index, err := db.SelectNullInt(
		"SELECT MAX(entryNum) FROM logEntries WHERE logID = ?",
		logID,
	)
	if err != nil {
		return 0, err
	}
	if index.Valid {
		return index.Int64, nil
	}
	return 0, nil
}
