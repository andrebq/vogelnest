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
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

type (
	// TweetLogWriter writes tweets to a boltdb
	TweetLogWriter struct {
		// directory where data is kept
		activefile string

		activedb *badger.DB

		entriesWritten prometheus.Counter
		bytesWritten   prometheus.Counter
	}

	// LogEntryKey represents a key from the log
	LogEntryKey struct {
		buf [1 + 8 + 1 + 8]byte
	}

	badgerLogger struct {
		zerolog.Logger
	}
)

var (
	// ErrClosed is sent when the user tries to write to a closed log
	ErrClosed = errors.New("already closed")

	bytesWrittenVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "bytesWritten",
		Namespace: "vogelnest",
		Subsystem: "tweetlogwriter",
	}, []string{"activeFile"})

	entriesWrittenVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "entriesWritten",
		Namespace: "vogelnest",
		Subsystem: "tweetlogwriter",
	}, []string{"activeFile"})
)

func init() {
	prometheus.MustRegister(bytesWrittenVec, entriesWrittenVec)
}

// NewLog takes a directory and creates one WAL file 15 minutes
func NewLog(dir string) (*TweetLogWriter, error) {
	now := time.Now().Truncate(time.Hour)
	activefile := filepath.Join(dir, "tweetlog", now.Format("2006-01-02_15"))
	err := os.MkdirAll(activefile, 0755)
	if err != nil {
		return nil, err
	}
	opts := badger.DefaultOptions(activefile)
	opts.Logger = &badgerLogger{log.Logger.With().Str("module", "tweet-log-writer").Str("db", filepath.Base(activefile)).Logger()}
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &TweetLogWriter{
		activefile: activefile,
		activedb:   db,

		entriesWritten: entriesWrittenVec.With(prometheus.Labels{"activeFile": filepath.Base(activefile)}),
		bytesWritten:   bytesWrittenVec.With(prometheus.Labels{"activeFile": filepath.Base(activefile)}),
	}, nil
}

// Close the underlying file, it is safe to call
// multiple times as only the first time will actually
// interact with the underlying fs
func (tl *TweetLogWriter) Close() error {
	if tl.activedb == nil {
		return nil
	}
	return tl.activedb.Close()
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
	totalBytes := float64(0)
	totalEntries := float64(0)
	for _, e := range entries {
		buf, err := proto.Marshal(e)
		if err != nil {
			return fmt.Errorf("unable to encode message: %w", err)
		}

		// wasting memory here, could re-use a temporary buffer
		// or sync.pool
		buf = snappy.Encode(nil, buf)

		var lek LogEntryKey
		lek.Set(now, e)
		err = bw.Set(lek.buf[:], buf)
		if err != nil {
			return fmt.Errorf("unable to add key to batch: %v", err)
		}
		totalBytes += float64(len(buf))
		totalEntries++
	}
	err := bw.Flush()
	if err != nil {
		return fmt.Errorf("unable to save data to disk: %v", err)
	}
	tl.bytesWritten.Add(totalBytes)
	tl.entriesWritten.Add(totalEntries)
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
