package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	cross "github.com/rolandshoemaker/cross-pollinator"

	"github.com/cactus/go-statsd-client/statsd"
)

type pollinator struct {
	logUpdateInterval time.Duration
	logWorkers        int
	logs              []*cross.Log
	entries           chan *cross.LogEntry

	db        *cross.Database
	dbWorkers int

	submissionRequests chan cross.SubmissionRequest
	wrMu               *sync.RWMutex
	pendingRequests    map[[32]byte]struct{} // hash of something... chain fp + log id?

	stats statsd.Statter
}

func (p *pollinator) getUpdates() {
	totalEntries := int64(0)
	for _, l := range p.logs {
		err := l.UpdateRemoteIndex()
		if err != nil {
			// log
			fmt.Println(err)
		}
		totalEntries += l.MissingEntries()
	}

	p.entries = make(chan *cross.LogEntry, totalEntries)
	wg := new(sync.WaitGroup)
	for _, l := range p.logs {
		wg.Add(1)
		go func(log *cross.Log) {
			defer wg.Done()
			err := log.GetNewEntries(p.entries)
			if err != nil {
				// log
				fmt.Println(err)
				return
			}
		}(l)
	}

	finishedProcessing := make(chan struct{})
	go func() {
		defer func() { finishedProcessing <- struct{}{} }()
		p.processEntries()
	}()
	wg.Wait()
	close(p.entries)
	<-finishedProcessing

	for _, l := range p.logs {
		err := l.UpdateLocalIndex()
		if err != nil {
			// log
			fmt.Println(err)
			continue
		}
	}
}

func (p *pollinator) processEntries() {
	wg := new(sync.WaitGroup)
	for i := 0; i < p.dbWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for e := range p.entries {
				s := time.Now()
				err := p.db.AddEntry(e)
				p.stats.TimingDuration("entries.entry-insert-latency", time.Since(s), 1.0)
				if err != nil {
					// log
					fmt.Println(err)
				}
				p.stats.Inc("entries.entries-inserted", 1, 1.0)
			}
		}()
	}
	wg.Wait()
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
	configFilename := "config.json"

	var config cross.Config
	contents, err := ioutil.ReadFile(configFilename)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = json.Unmarshal(contents, &config)
	if err != nil {
		fmt.Println(err)
		return
	}

	db := cross.NewDatabase(config.DatabaseURI)
	stats, err := statsd.NewClient(config.StatsdURI, "pollinator")
	if err != nil {
		fmt.Println(err)
		return
	}
	logs := make([]*cross.Log, len(config.Logs))
	for i, kl := range config.Logs {
		log, err := cross.NewLog(db, stats, kl)
		if err != nil {
			fmt.Println(err)
			return
		}
		logs[i] = log
	}

	fmt.Println(logs)
	p := pollinator{
		db:        db,
		dbWorkers: 15,
		stats:     stats,
		logs:      logs,
	}

	go p.getUpdates()
	for {
		fmt.Println(len(p.entries))
		time.Sleep(time.Millisecond * 250)
	}
}
