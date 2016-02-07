package cross

import (
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cactus/go-statsd-client/statsd"
	ct "github.com/google/certificate-transparency/go"
	ctClient "github.com/google/certificate-transparency/go/client"
)

type certificate struct {
	ID     int64
	Hash   []byte
	Offset int64
	Length int64
}

type certificateChain struct {
	ID                   int64
	Hash                 []byte
	CertIDs              []int64 `db:"-"`
	RootDN               string
	EntryType            ct.LogEntryType
	UnparseableComponent bool
	Logs                 map[string]struct{}
}

type LogEntry struct {
	ID       int64
	EntryNum int64
	LogID    []byte
	ChainID  int64
}

type Log struct {
	Name        string
	ID          []byte
	localIndex  int64
	remoteIndex int64

	uri    string
	client *ctClient.LogClient

	db       *Database
	certFile *byteFile
	stats    statsd.Statter

	validRoots map[string]struct{}
}

func NewLog(db *Database, stats statsd.Statter, kl KnownLog) (*Log, error) {
	name, uri := kl.Description, kl.URL
	if !strings.HasPrefix(uri, "http") {
		uri = "https://" + uri
	}
	pkBytes, err := base64.StdEncoding.DecodeString(kl.Key)
	if err != nil {
		return nil, err
	}
	id := sha256.Sum256(pkBytes)
	log := &Log{
		Name:       name,
		ID:         id[:],
		uri:        uri,
		client:     ctClient.New(uri),
		db:         db,
		stats:      stats,
		validRoots: make(map[string]struct{}),
	}
	err = log.populateRoots()
	if err != nil {
		return nil, err
	}
	err = log.UpdateLocalIndex()
	if err != nil {
		return nil, err
	}
	return log, nil
}

func (l *Log) UpdateLocalIndex() error {
	index, err := l.db.getCurrentLogIndex(l.ID)
	if err != nil {
		return err
	}
	l.localIndex = index
	return nil
}

func (l *Log) UpdateRemoteIndex() error {
	sth, err := l.client.GetSTH()
	if err != nil {
		return err
	}
	l.remoteIndex = int64(sth.TreeSize - 1)
	fmt.Println(l.Name, l.localIndex, l.remoteIndex)
	l.stats.Gauge(fmt.Sprintf("entries.remaining.%s", l.Name), l.remoteIndex-l.localIndex, 1.0)
	return nil
}

func (l *Log) MissingEntries() int64 {
	return l.remoteIndex - l.localIndex
}

func (l *Log) populateRoots() error {
	resp, err := http.Get(fmt.Sprintf("%s/ct/v1/get-roots", l.uri))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var encodedRoots struct {
		Certificates []string `json:"certificates"`
	}
	err = json.Unmarshal(body, &encodedRoots)
	if err != nil {
		return err
	}
	for _, encodedRoot := range encodedRoots.Certificates {
		rawCert, err := base64.StdEncoding.DecodeString(encodedRoot)
		if err != nil {
			// log
			fmt.Println(err)
			continue
		}
		root, err := x509.ParseCertificate(rawCert)
		if err != nil {
			// log
			fmt.Println(err)
			continue
		}
		subject := subjectToString(root.Subject)
		if subject == "" {
			// log + stat
			fmt.Println("weird")
			continue
		}
		l.validRoots[subject] = struct{}{}
	}
	return nil
}

func (l *Log) findRoot(chain []ct.ASN1Cert) (string, error) {
	for _, asnCert := range chain {
		cert, err := x509.ParseCertificate(asnCert)
		if err != nil {
			// badd
			fmt.Println(err)
			continue
		}
		issuer := subjectToString(cert.Issuer)
		if _, present := l.validRoots[issuer]; present {
			return issuer, nil
		}
		subject := subjectToString(cert.Subject)
		if _, present := l.validRoots[subject]; present {
			return subject, nil
		}
	}
	return "", fmt.Errorf("No suitable root found")
}

// func (l *Log) addCertificate(der []byte) (int64, error) {
// 	certFP := sha256.Sum256(der)
// 	if certID, err := l.db.GetCertificateID(certFP[:]); err == nil {
// 		return certID, nil
// 	}
// }

func (l *Log) addChain(e *ct.LogEntry) (int64, error) {
	var fullEntry [][]byte
	entryType := e.Leaf.TimestampedEntry.EntryType
	switch entryType {
	case ct.X509LogEntryType:
		fullEntry = append(fullEntry, []byte(e.Leaf.TimestampedEntry.X509Entry))
	case ct.PrecertLogEntryType:
		fullEntry = append(fullEntry, []byte(e.Leaf.TimestampedEntry.PrecertEntry.TBSCertificate))
	}
	for _, asnCert := range e.Chain {
		fullEntry = append(fullEntry, []byte(asnCert))
	}

	hasher := sha256.New()
	for _, c := range fullEntry {
		hasher.Write([]byte(c))
	}
	chainFP := hasher.Sum(nil)
	if chainID, err := l.db.GetChainID(chainFP); err == nil {
		return chainID, nil
	} else if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	var issuerChain []ct.ASN1Cert
	if len(e.Chain) == 0 {
		switch entryType {
		case ct.X509LogEntryType:
			issuerChain = []ct.ASN1Cert{e.Leaf.TimestampedEntry.X509Entry}
		case ct.PrecertLogEntryType:
			issuerChain = []ct.ASN1Cert{e.Leaf.TimestampedEntry.PrecertEntry.TBSCertificate}
		}
	} else {
		issuerChain = e.Chain
	}

	chain := &certificateChain{
		Hash:      chainFP,
		EntryType: entryType,
		Logs:      map[string]struct{}{fmt.Sprintf("%X", l.ID): struct{}{}},
	}

	rootDN, err := l.findRoot(issuerChain)
	if err != nil {
		// log
		fmt.Println(err)
		chain.UnparseableComponent = true
	}
	chain.RootDN = rootDN

	// for _, c := range fullEntry {
	// 	id, err := l.addCertificate(c)
	// 	if err != nil {
	// 		return 0, err
	// 	}
	// 	chain.CertIDs = append(chain.CertIDs, id)
	// }

	id, err := l.db.AddChain(chain)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (l *Log) processEntry(e *ct.LogEntry) error {
	entry := &LogEntry{EntryNum: e.Index, LogID: l.ID}
	chainID, err := l.addChain(e)
	if err != nil {
		return err
	}
	entry.ChainID = chainID
	err = l.db.AddEntry(entry)
	return err
}

func (l *Log) ProcessEntries(downloadedEntries chan ct.LogEntry) error {
	for e := range downloadedEntries {
		err := l.processEntry(&e)
		if err != nil {
			// log
			fmt.Println(err)
		}
	}
	return nil
}

func (l *Log) GetNewEntries(downloadedEntries chan ct.LogEntry) error {
	defer func() { close(downloadedEntries) }()
	if l.remoteIndex <= l.localIndex {
		// stat?
		return nil
	}

	addedUpTo := l.localIndex
	for addedUpTo < l.remoteIndex {
		max := addedUpTo + 2000
		if max > l.remoteIndex {
			max = l.remoteIndex
		}
		s := time.Now()
		entries, err := l.client.GetEntries(addedUpTo, max)
		l.stats.TimingDuration(fmt.Sprintf("entries.download-latency.%s", l.Name), time.Since(s), 1.0)
		if err != nil {
			// log && backoff?
			fmt.Println(l.Name, err)
			continue
		}
		fmt.Println(l.Name, len(entries))
		l.stats.Inc(fmt.Sprintf("entries.downloaded.%s", l.Name), int64(len(entries)), 1.0)
		for _, e := range entries {
			downloadedEntries <- e
			addedUpTo++
			l.stats.Inc(fmt.Sprintf("entries.processed.%s", l.Name), 1, 1.0)
			l.stats.Gauge(fmt.Sprintf("entries.remaining.%s", l.Name), l.remoteIndex-addedUpTo, 1.0)
		}
	}
	return nil
}

func (l *Log) Update() error {
	err := l.UpdateRemoteIndex()
	if err != nil {
		return err
	}
	bufferSize := 5000
	if actuallyMissing := l.MissingEntries(); actuallyMissing < 5000 {
		bufferSize = int(actuallyMissing)
	}
	entriesBuf := make(chan ct.LogEntry, bufferSize)
	go func() {
		err := l.GetNewEntries(entriesBuf)
		if err != nil {
			// log
			fmt.Println(err)
		}
	}()
	err = l.ProcessEntries(entriesBuf)
	if err != nil {
		return err
	}
	return l.UpdateLocalIndex()
}

type SubmissionRequest struct {
	entryNum int64
	srcLog   *Log
	dstLog   *Log
}
