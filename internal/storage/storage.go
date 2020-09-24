package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/andrebq/vogelnest/internal/schema"
	"github.com/dgraph-io/badger"
	"google.golang.org/protobuf/proto"
)

type (
	// TweetLogWriter writes tweets to a boltdb
	TweetLogWriter struct {
		// directory where data is kept
		activefile string

		activedb *badger.DB
	}

	// LogEntryKey represents a key from the log
	LogEntryKey struct {
		buf [1 + 8 + 1 + 8]byte
	}
)

var (
	// ErrClosed is sent when the user tries to write to a closed log
	ErrClosed = errors.New("already closed")
)

// NewLog takes a directory and creates one WAL file 15 minutes
func NewLog(dir string) (*TweetLogWriter, error) {
	err := os.MkdirAll(dir, 0644)
	if err != nil {
		return nil, err
	}
	now := time.Now().Truncate(time.Hour)
	activefile := filepath.Join(dir, fmt.Sprintf("tweetlog-%v", now.Format("2006-01-02_15")))
	db, err := badger.Open(badger.DefaultOptions(activefile))
	if err != nil {
		return nil, err
	}
	return &TweetLogWriter{
		activefile: activefile,
		activedb:   db,
	}, nil
}

// Close the underlying file, it is safe to call
// multiple times as only the first time will actually
// interact with the underlying fs
func (tl *TweetLogWriter) Close() error {
	if tl.activedb == nil {
		return nil
	}
	return nil
}

// Append an entry to the log
func (tl *TweetLogWriter) Append(entries ...*schema.Tweet) error {
	if tl.activedb == nil {
		return ErrClosed
	}

	now := time.Now().Truncate(time.Minute * 10).Unix()

	bw := tl.activedb.NewWriteBatch()
	defer func() {
		if bw != nil {
			bw.Cancel()
		}
	}()
	for _, e := range entries {
		buf, err := proto.Marshal(e)
		if err != nil {
			return fmt.Errorf("unable to encode message: %w", err)
		}
		var lek LogEntryKey
		lek.Set(now, e)
		err = bw.Set(lek.buf[:], buf)
		if err != nil {
			return fmt.Errorf("unable to add key to batch: %v", err)
		}
	}
	err := bw.Flush()
	if err != nil {
		return fmt.Errorf("unable to save data to disk: %v", err)
	}
	bw = nil
	return nil
}

// Set the content of this key
func (l *LogEntryKey) Set(moment int64, e *schema.Tweet) {
	l.buf[0] = byte('l')
	binary.BigEndian.PutUint64(l.buf[1:], uint64(moment))
	l.buf[9] = byte('/')
	binary.BigEndian.PutUint64(l.buf[1+8+1:], uint64(e.Id))
}
