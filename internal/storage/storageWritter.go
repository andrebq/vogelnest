package storage

import (
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/andrebq/vogelnest/internal/schema"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
)

type (
	flusher interface {
		io.Writer
		Flush() error
	}

	noopFlusher struct {
		io.Writer
	}

	// TweetLogWriter writes tweets to a boltdb
	TweetLogWriter struct {
		activefile   *os.File
		activestream flusher

		entriesWritten prometheus.Counter
		bytesWritten   prometheus.Counter
		logctx         zerolog.Logger
	}

	// LogEntryKey represents a key from the log
	LogEntryKey struct {
		buf [1 + 8 + 1 + 8]byte
	}

	badgerLogger struct {
		zerolog.Logger
	}

	tlvTag byte
)

const (
	tweeetTag = tlvTag(0)
)

var (
	// ErrClosed is sent when the user tries to write to a closed log
	ErrClosed = errors.New("already closed")

	bytesWrittenVec = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "bytesWritten",
		Namespace: "vogelnest",
		Subsystem: "tweetlogwriter",
	})

	entriesWrittenVec = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "entriesWritten",
		Namespace: "vogelnest",
		Subsystem: "tweetlogwriter",
	})
)

func init() {
	prometheus.MustRegister(bytesWrittenVec, entriesWrittenVec)
}

// NewLog takes a directory and creates one WAL file 15 minutes
func NewLog(dir string, trunc time.Duration) (*TweetLogWriter, error) {
	if trunc < time.Minute {
		return nil, errors.New("truncation should be at least on a minute basis")
	}
	activefilePath := filepath.Join(dir, "tweetlog.json")
	err := os.MkdirAll(filepath.Dir(activefilePath), 0755)
	if err != nil {
		return nil, err
	}
	activefile, err := os.OpenFile(activefilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	// compressWriter, err := gzip.NewWriterLevel(activefile, 9)
	// if err != nil {
	// 	activefile.Close()
	// 	return nil, err
	// }
	return &TweetLogWriter{
		activefile:   activefile,
		activestream: noopFlusher{activefile},
		logctx:       log.Logger.With().Str("system", "tweet-log-writer").Str("active-file", filepath.Base(activefilePath)).Logger(),

		entriesWritten: entriesWrittenVec,
		bytesWritten:   bytesWrittenVec,
	}, nil
}

// Pack will close this stream and rename it by adding '.packed',
// this indicates that this file should be used for read-only operations
//
// After Pack is called this log is invalid.
func (tl *TweetLogWriter) Pack() error {
	if tl.activefile == nil {
		return ErrClosed
	}
	// no need to flush or sync
	// as those operations are executed
	// as part of Append call
	fp := tl.activefile.Name()
	err := tl.Close()
	if err != nil {
		tl.markCorrupted()
		return err
	}
	return tl.doPack(fp)
}

func (tl *TweetLogWriter) doPack(oldfile string) error {
	chunkid := time.Now().Format("20060102_150405")
	packingfile := filepath.Join(filepath.Dir(oldfile), fmt.Sprintf("tweets-%v.packing", chunkid))
	packedfile := filepath.Join(filepath.Dir(oldfile), fmt.Sprintf("tweets-%v.gz", chunkid))

	err := os.Rename(oldfile, packingfile)
	if err != nil {
		return err
	}
	// at this point it is safe to finish this process in another goroutine
	// but let's not worry about that now
	packingfd, err := os.Open(packingfile)
	if err != nil {
		return err
	}
	defer packingfd.Close()

	packedfd, err := os.OpenFile(packedfile, os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		log.Error().Err(err).Msg("wtf is going on!")
		return err
	}
	defer packedfd.Close()

	gz, err := gzip.NewWriterLevel(packedfd, gzip.BestCompression)
	if err != nil {
		log.Error().Err(err).Msg("wtf is going on!")
		return err
	}
	buf := make([]byte, 1024)
	for {
		n, err := packedfd.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			println("wtf!!!!!!!!!!!!!!!!!!!!!!!!1")
			return err
		}
		_, err = gz.Write(buf[:n])
		if err != nil {
			println("reallly!!!!")
			return err
		}
	}
	// _, err = io.Copy(gz, packingfd)
	// if err != nil {
	// 	return err
	// }
	err = gz.Flush()
	if err != nil {
		return err
	}
	err = gz.Close()
	if err != nil {
		return err
	}
	return packedfd.Sync()
}

// Close the underlying file, it is safe to call
// multiple times as only the first time will actually
// interact with the underlying fs
func (tl *TweetLogWriter) Close() error {
	if tl.activefile == nil {
		return nil
	}
	err := tl.activefile.Close()
	tl.activefile = nil
	return err
}

// Append an entry to the log
func (tl *TweetLogWriter) Append(entries ...*schema.Tweet) error {
	if tl.activefile == nil {
		return ErrClosed
	}
	totalBytes := float64(0)
	totalEntries := float64(0)
	for _, e := range entries {
		_, err := io.WriteString(tl.activestream, "\n")
		if err != nil {
			tl.logctx.Error().Err(err).Str("action", "append").Str("subAction", "newline").Send()
			continue
		}
		buf, err := protojson.Marshal(e)
		if err != nil {
			tl.logctx.Error().Err(err).Str("action", "append").Str("subAction", "protoencoding").Send()
			continue
		}

		n, err := tl.activestream.Write(buf)
		if err != nil {
			tl.logctx.Error().Err(err).Str("action", "append").Str("subAction", "writebuf").Send()
			continue
		}

		totalBytes += float64(n + 1 /* newline */)
		totalEntries++
	}
	err := tl.activestream.Flush()
	if err != nil {
		tl.logctx.Error().Err(err).Str("action", "append").Str("subAction", "flushStream").Send()
		tl.markCorrupted()
		return err
	}
	err = tl.activefile.Sync()
	if err != nil {
		// cannot safely process the file anymore
		// should this cause a panic?
		tl.markCorrupted()
		return fmt.Errorf("unable to save data to disk, system in a corrupted state: %v", err)
	}
	tl.bytesWritten.Add(totalBytes)
	tl.entriesWritten.Add(totalEntries)
	return nil
}

func (tl *TweetLogWriter) markCorrupted() {
	path := tl.activefile.Name()
	err := tl.Close()
	if err != nil {
		tl.logctx.Error().Err(err).Str("action", "mark-corrupted").Msg("Error while closing file")
	}
	newpath := fmt.Sprintf("%v.corrupted", path)
	err = os.Rename(path, newpath)
	if err != nil {
		tl.logctx.Error().Err(err).Str("action", "mark-corrupted").Str("new-path", newpath).Msg("Unable to rename file, further dataloss might happen")
	}
}

// Set the content of this key
func (l *LogEntryKey) Set(moment int64, e *schema.Tweet) {
	l.buf[0] = byte('l')
	binary.BigEndian.PutUint64(l.buf[1:], uint64(moment))
	l.buf[9] = byte('/')
	binary.BigEndian.PutUint64(l.buf[1+8+1:], uint64(e.Id))
}

func tlvEncode(out io.Writer, tag tlvTag, buf []byte) (int, error) {
	var scratch [5]byte

	scratch[0] = byte(tag)
	total := int(0)
	var n int
	var err error

	if n, err = out.Write(scratch[:1]); err != nil {
		return n, err
	}
	total += n

	var aux [binary.MaxVarintLen64]byte
	sz := binary.PutVarint(aux[:], int64(len(buf)))
	_, err = out.Write(aux[:sz])
	if err != nil {
		return n, err
	}

	if n, err = out.Write(buf); err != nil {
		return total + n, err
	}
	return total + n, nil
}

func (n noopFlusher) Flush() error { return nil }
