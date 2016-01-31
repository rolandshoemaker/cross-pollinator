package main

import (
	"crypto"
	"fmt"
	"sync"
	"time"

	cross "github.com/rolandshoemaker/cross-pollinator"

	"github.com/cactus/go-statsd-client/statsd"
)

type pollinator struct {
	logUpdateInterval time.Duration
	logWorkers        int
	logs              map[[32]byte]*cross.Log
	entries           chan *cross.LogEntry

	db *cross.Database

	submissionRequests chan cross.SubmissionRequest
	wrMu               *sync.RWMutex
	pendingRequests    map[[32]byte]struct{} // hash of something... chain fp + log id?

	stats statsd.Statter
}

func (p *pollinator) getUpdates() {
	wg := new(sync.WaitGroup)
	for _, l := range p.logs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := l.GetEntries(p.entries)
			if err != nil {
				// log
				fmt.Println(err)
				return
			}
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
	db := cross.NewDatabase()
	var pk crypto.PublicKey
	logURI := "https://ct.googleapis.com/pilot"
	log, err := cross.NewLog(db, "log", logURI, pk)
	if err != nil {
		fmt.Println(err)
		return
	}

	p := pollinator{
		db:      db,
		entries: make(chan *cross.LogEntry, 1000000),
		logs: map[[32]byte]*cross.Log{
			id: log,
		},
	}

	go p.getUpdates()
	go func() {
		for {
			fmt.Println(len(p.entries))
			time.Sleep(time.Millisecond * 250)
		}
	}()
	for e := range p.entries {
		err := p.db.AddEntry(e)
		if err != nil {
			fmt.Println(err)
		}
	}
}
