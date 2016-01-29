package cross

import (
	ct "github.com/letsencrypt/certificate-transparency/go"
	ctClient "github.com/letsencrypt/certificate-transparency/go/client"
)

type validRoot struct {
	DN    string
	LogID [32]byte
}

type logEntry struct {
	Hash     [32]byte
	RootDN   string
	EntryNum int64
	LogID    [32]byte
}

type log struct {
	Name        string
	ID          [32]byte
	LocalIndex  int64
	RemoteIndex int64

	client ctClient.LogClient
}

type progress struct {
	SourceLog      [32]byte `db:"srcLog"`
	DestinationLog [32]byte `db:"dstLog"`
	CurrentIndex   int64
}

type submissionRequest struct {
	entryNum int64
	srcLog   *log
	dstLog   *log
}
