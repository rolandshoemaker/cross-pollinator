package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	cross "github.com/rolandshoemaker/cross-pollinator"

	"github.com/cactus/go-statsd-client/statsd"
	// ct "github.com/google/certificate-transparency/go"
)

type pollinator struct {
	logUpdateInterval time.Duration
	logs              []*cross.Log

	db        *cross.Database
	dbWorkers int

	submissionRequests chan cross.SubmissionRequest
	wrMu               *sync.RWMutex
	pendingRequests    map[[32]byte]struct{} // hash of something... chain fp + log id?

	stats statsd.Statter
}

func (p *pollinator) getUpdates() {
	wg := new(sync.WaitGroup)
	for _, l := range p.logs {
		wg.Add(1)
		go func(log *cross.Log) {
			defer wg.Done()
			err := log.Update()
			if err != nil {
				// log
				fmt.Println(err)
			}
		}(l)
	}
	wg.Wait()
}

func (p *pollinator) findViableEntries() {

}

func (p *pollinator) submitEntries() {

}

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

	stats, err := statsd.NewClient(config.StatsdURI, "pollinator")
	if err != nil {
		fmt.Println(err)
		return
	}
	db, err := cross.NewDatabase(config.DatabaseURI, stats)
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

	p := pollinator{
		db:        db,
		dbWorkers: 250,
		stats:     stats,
		logs:      logs,
	}
	p.getUpdates()
}
