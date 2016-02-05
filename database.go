package cross

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
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

	dbMap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}, TypeConverter: typeConverter{}}
	dbMap.AddTableWithName(LogEntry{}, "logEntries").SetKeys(true, "ID")
	dbMap.AddTableWithName(certificate{}, "certificates").SetKeys(true, "ID")
	// dbMap.TraceOn("SQL: ", log.New(os.Stdout, "", 5))
	return &Database{dbMap}
}

func (db *Database) getCertificateID(hash []byte) (int64, error) {
	var id int64
	err := db.DbMap.SelectOne(
		&id,
		"SELECT id FROM certificates WHERE hash = ?",
		hash,
	)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (db *Database) addCertificate(der []byte) (int64, error) {
	hash := sha256.Sum256(der)
	if id, err := db.getCertificateID(hash[:]); err != nil && err != sql.ErrNoRows {
		return 0, err
	} else if err == nil {
		return id, nil
	}
	cert := &certificate{
		Hash: hash[:],
		DER:  der,
	}
	err := db.DbMap.Insert(cert)
	if err != nil && !strings.HasPrefix(err.Error(), "Error 1062: Duplicate entry") {
		return 0, err
	}
	return cert.ID, nil
}

func (db *Database) AddEntry(e *LogEntry) error {
	IDs := []int64{}
	for _, der := range e.Chain {
		id, err := db.addCertificate(der)
		if err != nil {
			return err
		}
		IDs = append(IDs, id)
	}
	e.ChainIDs = IDs
	err := db.DbMap.Insert(e)
	if err != nil && !strings.HasPrefix(err.Error(), "Error 1062: Duplicate entry") {
		return err
	}
	return nil
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

type typeConverter struct{}

func (tc typeConverter) ToDb(val interface{}) (interface{}, error) {
	switch t := val.(type) {
	case []int64:
		jsonBytes, err := json.Marshal(t)
		if err != nil {
			return nil, err
		}
		return string(jsonBytes), nil
	default:
		return val, nil
	}
}

func (tc typeConverter) FromDb(target interface{}) (gorp.CustomScanner, bool) {
	switch target.(type) {
	case *[]int:
		binder := func(holder, target interface{}) error {
			s, ok := holder.(*string)
			if !ok {
				return errors.New("FromDb: Unable to convert *string")
			}
			b := []byte(*s)
			return json.Unmarshal(b, target)
		}
		return gorp.CustomScanner{Holder: new(string), Target: target, Binder: binder}, true
	default:
		return gorp.CustomScanner{}, false
	}
}
