package cross

import (
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	ct "github.com/google/certificate-transparency/go"
	ctClient "github.com/google/certificate-transparency/go/client"
)

type entryContent struct {
	Hash    []byte
	Content []byte
}

type LogEntry struct {
	ID                   int64
	SubmissionHash       []byte
	RootDN               string
	EntryNum             int64
	LogID                []byte
	UnparseableComponent bool
	EntryType            ct.LogEntryType

	Submission []byte `db:"-"`
}

type Log struct {
	name        string
	id          []byte
	localIndex  int64
	remoteIndex int64

	uri    string
	client *ctClient.LogClient

	db *Database

	validRoots map[string]struct{}
}

func NewLog(db *Database, name, uri string, pk crypto.PublicKey) (*Log, error) {
	var id []byte
	log := &Log{
		name:       name,
		id:         id,
		uri:        uri,
		client:     ctClient.New(uri),
		db:         db,
		validRoots: make(map[string]struct{}),
	}
	err := log.populateRoots()
	if err != nil {
		return nil, err
	}
	err = log.updateLocalIndex()
	if err != nil {
		return nil, err
	}
	err = log.UpdateRemoteIndex()
	if err != nil {
		return nil, err
	}
	return log, nil
}

func (l *Log) updateLocalIndex() error {
	index, err := l.db.getCurrentLogIndex(l.id)
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
	l.remoteIndex = int64(sth.TreeSize)
	return nil
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

func (l *Log) processEntry(e ct.LogEntry) *LogEntry {
	chain := []ct.ASN1Cert{}
	fullEntry := []byte{}
	entryType := e.Leaf.TimestampedEntry.EntryType
	switch entryType {
	case ct.X509LogEntryType:
		fullEntry = append(fullEntry, []byte(e.Leaf.TimestampedEntry.X509Entry)...)
	case ct.PrecertLogEntryType:
		fullEntry = append(fullEntry, []byte(e.Leaf.TimestampedEntry.PrecertEntry.TBSCertificate)...)
	}
	for _, asnCert := range e.Chain {
		fullEntry = append(fullEntry, []byte(asnCert)...)
	}
	if len(e.Chain) == 0 {
		switch entryType {
		case ct.X509LogEntryType:
			chain = append(chain, e.Leaf.TimestampedEntry.X509Entry)
		case ct.PrecertLogEntryType:
			chain = append(chain, e.Leaf.TimestampedEntry.PrecertEntry.TBSCertificate)
		}
	} else {
		chain = e.Chain
	}
	unparseable := false
	rootDN, err := l.findRoot(chain)
	if err != nil {
		// log
		fmt.Println(err)
		unparseable = true
	}
	contentHash := sha256.Sum256(fullEntry)
	return &LogEntry{
		SubmissionHash:       contentHash[:],
		RootDN:               rootDN,
		EntryNum:             e.Index,
		LogID:                l.id,
		Submission:           fullEntry,
		EntryType:            entryType,
		UnparseableComponent: unparseable,
	}
}

func (l *Log) GetEntries(processed chan *LogEntry) error {
	err := l.UpdateRemoteIndex()
	if err != nil {
		return err
	}
	fmt.Println(l.localIndex, l.remoteIndex)
	if l.remoteIndex == l.localIndex {
		// stat?
		return nil
	}

	addedUpTo := l.localIndex
	for addedUpTo < l.remoteIndex {
		max := addedUpTo + 1000
		if max > l.remoteIndex {
			max = l.remoteIndex
		}
		entries, err := l.client.GetEntries(addedUpTo, max)
		if err != nil {
			// log && backoff?
			fmt.Println(err)
			continue
		}
		for _, e := range entries {
			if addedUpTo != 0 && e.Index == 0 {
				break // ct client bug
			}
			processed <- l.processEntry(e)
			addedUpTo++
		}
	}
	return nil
}

type SubmissionRequest struct {
	entryNum int64
	srcLog   *Log
	dstLog   *Log
}
