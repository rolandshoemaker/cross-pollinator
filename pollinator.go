package cross

import (
	"sync"

	"github.com/cactus/go-statsd-client/statsd"
	"gopkg.in/gorp.v1"
)

type pollinator struct {
	cursorCheckInterval time.Duration
	logWorkers          int
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
func (p *pollinator) run() {

}

func (p *pollinator) getEntries() {
	wg := new(sync.WaitGroup)
	for _, l := range p.logs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := l.updateRemoteIndex()
			if err != nil {
				// log
				return
			}
			if l.RemoteIndex == l.LocalIndex {
				// stat?
				return
			}
			ranges := make(chan [2]int)
			for _, r := range entryRanges(l.CurrentIndex, l.RemoteIndex) {
				ranges <- r
			}
			close(ranges)
			wg := new(sync.WaitGroup)
			for i := 0; i < l.logWorkers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for r := range ranges {
						entries, err := l.client.GetEntries(r[0], r[1])
						if err != nil {
							// ughhhh retry probably.. hate this stuff :/
							continue
						}
						for _, e := range entries {
							// actually extract stuff
							p.entries <- logEntry{}
						}
					}
				}()
			}
			wg.Wait()
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
