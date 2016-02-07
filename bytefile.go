package cross

import (
	"fmt"
	"os"
	"sync"
)

type byteFile struct {
	*os.File
	mu            *sync.RWMutex
	currentOffset int64
}

func newByteFile(filename string) (*byteFile, error) {
	f, err := os.OpenFile(filename, 0, os.ModePerm) // FIX FLAG
	if err != nil {
		return nil, err
	}
	fs, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return &byteFile{f, new(sync.RWMutex), fs.Size()}, nil
}

func (bf *byteFile) readSection(offset, length int64) ([]byte, error) {
	buf := make([]byte, length)
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	r, err := bf.ReadAt(buf, offset)
	if err != nil {
		return nil, err
	} else if int64(r) != length {
		return nil, fmt.Errorf("Didn't read full section, only got %d of %d bytes", r, length)
	}
	return buf, nil
}

func (bf *byteFile) writeSection(stuff []byte) (int64, error) {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	offset := bf.currentOffset
	_, err := bf.WriteAt(stuff, offset)
	if err != nil {
		return 0, err
	}
	bf.currentOffset += int64(len(stuff))
	return offset, nil
}
