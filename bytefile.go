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

func NewByteFile(f *os.File) (*byteFile, error) {
	fs, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return &byteFile{f, new(sync.RWMutex), fs.Size()}, nil
}

func (bf *byteFile) ReadSection(offset, length int64) ([]byte, error) {
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

func (bf *byteFile) WriteSection(stuff []byte) (int64, error) {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	offset := bf.currentOffset
	w, err := bf.WriteAt(stuff, offset)
	if err != nil {
		return 0, err
	}
	if w != len(stuff) {
		return 0, fmt.Errorf("Partial write, expected %d, only wrote %d", len(stuff), w)
	}
	bf.currentOffset += int64(len(stuff))
	return offset, nil
}
