package cross

import (
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
	Name        string
	ID          []byte
	LocalIndex  int64
	RemoteIndex int64

	URI    string
	Client *ctClient.LogClient

	ValidRoots map[string]struct{}
}

func (l *Log) PopulateRoots() {
	resp, err := http.Get(fmt.Sprintf("%s/ct/v1/get-roots", l.URI))
	if err != nil {
		// fmt.Fprintf(os.Stderr, "failed to get CT log roots: %s\n", err)
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// fmt.Fprintf(os.Stderr, "failed to read CT log roots response: %s\n", err)
		fmt.Println(err)
		return
	}
	var encodedRoots struct {
		Certificates []string `json:"certificates"`
	}
	err = json.Unmarshal(body, &encodedRoots)
	if err != nil {
		// fmt.Fprintf(os.Stderr, "failed to parse CT log roots response: %s\n", err)
		fmt.Println(err)
		return
	}
	for _, encodedRoot := range encodedRoots.Certificates {
		rawCert, err := base64.StdEncoding.DecodeString(encodedRoot)
		if err != nil {
			fmt.Println(err)
			continue
		}
		root, err := x509.ParseCertificate(rawCert)
		if err != nil {
			fmt.Println(err)
			continue
		}
		subject := subjectToString(root.Subject)
		if subject == "" {
			fmt.Println("weird")
			continue
		}
		fmt.Println(subject)
		l.ValidRoots[subject] = struct{}{}
	}
}

func (l *Log) FindRoot(chain []ct.ASN1Cert) (string, error) {
	for _, asnCert := range chain {
		cert, err := x509.ParseCertificate(asnCert)
		if err != nil {
			// badd
			fmt.Println(err)
			continue
		}
		issuer := subjectToString(cert.Issuer)
		if _, present := l.ValidRoots[issuer]; present {
			return issuer, nil
		}
		subject := subjectToString(cert.Subject)
		if _, present := l.ValidRoots[subject]; present {
			return subject, nil
		}
	}
	return "", fmt.Errorf("No suitable root found")
}

func (l *Log) UpdateRemoteIndex() error {
	sth, err := l.Client.GetSTH()
	if err != nil {
		return err
	}
	l.RemoteIndex = int64(sth.TreeSize)
	return nil
}

type progress struct {
	SourceLog      [32]byte `db:"srcLog"`
	DestinationLog [32]byte `db:"dstLog"`
	CurrentIndex   int64
}

type SubmissionRequest struct {
	entryNum int64
	srcLog   *Log
	dstLog   *Log
}
