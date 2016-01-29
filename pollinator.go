package cross

import (
	"sync"

	"github.com/cactus/go-statsd-client/statsd"
	"gopkg.in/gorp.v1"
)

type pollinator struct {
	cursorCheckInterval time.Duration
	logs                map[[32]byte]log
	entries             chan logEntry

	db gorp.DbMap

	submissionRequests chan submissionRequest
	wrMu               *sync.RWMutex
	waitingRequests    map[[32]byte]struct{}

	stats statsd.Statter
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

func (p *pollinator) getEntries() {

}

func (p *pollinator) processEntries() {

}

func (p *pollinator) findViableEntries() {

}

func (p *pollinator) submitEntries() {

}
