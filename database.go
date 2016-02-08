package cross

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
	// "strings"
	// "log"
	// "os"

	"github.com/cactus/go-statsd-client/statsd"
	_ "github.com/lib/pq"
	"gopkg.in/gorp.v1"
)

type Database struct {
	*gorp.DbMap
	stats statsd.Statter
}

func NewDatabase(dbURI string, stats statsd.Statter) (*Database, error) {
	db, err := sql.Open("postgres", dbURI)
	if err != nil {
		return nil, err
	}
	db.SetMaxIdleConns(100)

	dbMap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}, TypeConverter: typeConverter{}}
	dbMap.AddTableWithName(LogEntry{}, "log_entries").SetKeys(true, "ID")
	dbMap.AddTableWithName(certificateChain{}, "chains").SetKeys(true, "ID")
	dbMap.AddTableWithName(certificate{}, "certificates").SetKeys(true, "ID")
	// dbMap.TraceOn("SQL: ", log.New(os.Stdout, "", 5))
	return &Database{dbMap, stats}, nil
}

func (db *Database) getCertificateID(hash []byte) (int64, error) {
	var id int64
	err := db.DbMap.SelectOne(
		&id,
		"SELECT id FROM certificates WHERE hash = $1",
		hash,
	)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	return id, nil
}

func (db *Database) GetChainID(hash []byte) (int64, error) {
	var id int64
	err := db.DbMap.SelectOne(
		&id,
		"SELECT id FROM chains WHERE hash = $1",
		hash,
	)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (db *Database) AddCertificate(hash []byte, offset, length int64) (int64, error) {
	if id, err := db.getCertificateID(hash); err != nil && err != sql.ErrNoRows {
		return 0, err
	} else if err == nil {
		return id, nil
	}
	cert := &certificate{
		Hash:   hash[:],
		Offset: offset,
		Length: length,
	}
	err := db.DbMap.Insert(cert)
	if err != nil {
		return 0, err
	}
	return cert.ID, nil
}

func (db *Database) AddChain(chain *certificateChain) (int64, error) {
	if id, err := db.GetChainID(chain.Hash); err != nil && err != sql.ErrNoRows {
		return 0, err
	} else if err == nil {
		return id, nil
	}
	s := time.Now()
	err := db.Insert(chain)
	db.stats.TimingDuration("db.inserts.chains", time.Since(s), 1.0)
	if err != nil {
		return 0, err
	}
	return chain.ID, nil
}

func (db *Database) AddEntry(e *LogEntry) error {
	s := time.Now()
	err := db.DbMap.Insert(e)
	db.stats.TimingDuration("db.inserts.entries", time.Since(s), 1.0)
	if err != nil {
		return err
	}
	return nil
}

func (db *Database) getCurrentLogIndex(logID []byte) (int64, error) {
	index, err := db.SelectNullInt(
		"SELECT MAX(entry_num) FROM log_entries WHERE log_id = $1",
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
	case []int64, map[string]struct{}:
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
	case *[]int, *map[string]struct{}:
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
