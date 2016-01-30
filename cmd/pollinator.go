package main

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	cross "github.com/rolandshoemaker/cross-pollinator"

	"github.com/cactus/go-statsd-client/statsd"
	ct "github.com/google/certificate-transparency/go"
	ctClient "github.com/google/certificate-transparency/go/client"
	// ctX509 "github.com/google/certificate-transparency/go/x509"
	"gopkg.in/gorp.v1"
)

type pollinator struct {
	logUpdateInterval time.Duration
	logWorkers        int
	logs              map[[32]byte]cross.Log
	entries           chan cross.LogEntry

	db *gorp.DbMap

	submissionRequests chan cross.SubmissionRequest
	wrMu               *sync.RWMutex
	pendingRequests    map[[32]byte]struct{} // hash of something... chain fp + log id?

	stats statsd.Statter
}

func (p *pollinator) retrieveEntries(l cross.Log) {
	addedUpTo := l.LocalIndex
	for addedUpTo < l.RemoteIndex {
		max := addedUpTo + 1000
		if max > l.RemoteIndex {
			max = l.RemoteIndex
		}
		entries, err := l.Client.GetEntries(addedUpTo, max)
		if err != nil {
			// log && backoff?
			fmt.Println(err)
			continue
		}
		for _, e := range entries {
			if addedUpTo != 0 && e.Index == 0 {
				break
			}
			chain := []ct.ASN1Cert{}
			fullEntry := []byte{}
			var entryType ct.LogEntryType
			entryType = e.Leaf.TimestampedEntry.EntryType
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
			rootDN, err := l.FindRoot(chain)
			if err != nil {
				// log
				fmt.Println(err)
				unparseable = true
			}
			contentHash := sha256.Sum256(fullEntry)
			entry := cross.LogEntry{
				SubmissionHash:       contentHash[:],
				RootDN:               rootDN,
				EntryNum:             e.Index,
				LogID:                l.ID,
				Submission:           fullEntry,
				EntryType:            entryType,
				UnparseableComponent: unparseable,
			}
			p.entries <- entry
			addedUpTo++
		}
	}
}

func (p *pollinator) getEntries() {
	wg := new(sync.WaitGroup)
	for _, l := range p.logs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := l.UpdateRemoteIndex()
			if err != nil {
				// log
				fmt.Println(err)
				return
			}
			fmt.Println(l.LocalIndex, l.RemoteIndex)
			if l.RemoteIndex == l.LocalIndex {
				// stat?
				return
			}
			p.retrieveEntries(l)
		}()
	}
	wg.Wait()
}

func (p *pollinator) processEntries() {

}

func (p *pollinator) findViableEntries() {

}

func (p *pollinator) submitEntries() {

}

// getEntries() -> p.entries concurrently for each log
//  |
//  v
// processEntries() p.entries -> into the database
//
// findViableEntries() -> p.submissionRequests
//  |
//  v
// submitEntries() p.submissionRequests ->
func (p *pollinator) run() {

}

func main() {
	id := [32]byte{255}
	logURI := "https://ct.googleapis.com/pilot"
	p := pollinator{
		db:      cross.InitDBMap(),
		entries: make(chan cross.LogEntry, 1000000),
		logs: map[[32]byte]cross.Log{
			id: {
				ID:         id[:],
				Client:     ctClient.New(logURI),
				URI:        logURI,
				LocalIndex: 8000000,
				ValidRoots: make(map[string]struct{}),
			},
		},
	}
	for _, l := range p.logs {
		l.PopulateRoots()
	}
	go p.getEntries()
	go func() {
		for {
			fmt.Println(len(p.entries))
			time.Sleep(time.Millisecond * 250)
		}
	}()
	for e := range p.entries {
		if e.UnparseableComponent {
			fmt.Println("one here")
		}
		err := cross.AddEntry(p.db, &e)
		if err != nil {
			fmt.Println(err)
		}
	}
}
