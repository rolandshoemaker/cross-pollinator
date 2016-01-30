package cross

import (
	"database/sql"
	// "log"
	// "os"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/gorp.v1"
)

func InitDBMap() *gorp.DbMap {
	db, err := sql.Open("mysql", "pollinator@/pollinator")
	if err != nil {
		panic(err)
	}

	dbMap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}
	dbMap.AddTableWithName(LogEntry{}, "logEntries").SetKeys(true, "ID")
	dbMap.AddTableWithName(entryContent{}, "submissionContents").SetKeys(false, "Hash")
	// dbMap.TraceOn("SQL: ", log.New(os.Stdout, "", 5))
	return dbMap
}

func addEntryContent(db *gorp.DbMap, c *entryContent) error {
	return db.Insert(c)
}

func AddEntry(db *gorp.DbMap, e *LogEntry) error {
	err := db.Insert(e)
	if err != nil {
		return err
	}
	content := &entryContent{
		Hash:    e.SubmissionHash,
		Content: e.Submission,
	}
	return addEntryContent(db, content)
}
