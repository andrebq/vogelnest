package trail

import (
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"time"
)

type (
	// segment is used to compose a trail, segments can be read-only
	// or write-only.
	segment struct {
		readonly    bool
		closed      bool
		datawritten bool
		packed      bool
		output      segmentOutput
		input       interface {
			io.Reader
			io.ByteReader
		}
		file segmentFile
		sum  hash.Hash32
	}

	updateSum struct {
		sum hash.Hash
		out io.Writer
	}

	closedReader struct{}

	failedReader struct {
		err error
	}

	entryReader struct {
		buf []byte
	}

	segmentOutput interface {
		io.Writer
		Close() error
		Flush() error
	}

	segmentFile interface {
		Sync() error
		Close() error
		Name() string
	}
)

var (
	koopmanTable = crc32.MakeTable(crc32.Koopman)
)

func (s *segment) packAndClose() error {
	if s.packed {
		return nil
	}
	s.packed = true
	name := s.file.Name()
	err := s.Close()
	if err != nil {
		return err
	}
	if !s.datawritten {
		return os.Remove(name)
	}
	currentSecond := time.Now().UTC().Truncate(time.Second)
	millis := time.Now().UTC().Sub(currentSecond).Milliseconds()
	now := fmt.Sprintf("%v_%x_%x.segment.gz", currentSecond.Format("20060102_150405"), millis, s.sum.Sum32())
	newname := filepath.Join(filepath.Dir(name), now)
	return os.Rename(name, newname)
}

func (s *segment) append(entry segmentEntry) (int64, error) {
	if s.closed {
		return 0, ErrClosed
	}
	s.datawritten = true
	if s.sum == nil {
		s.sum = crc32.New(koopmanTable)
	}
	return entry.WriteTo(updateSum{s.sum, s.output})
}

// NextEntry returns a reader which can be used to read the next entry
// callers shouldn't held to this reader longer than they held a Segment.
//
// A call to NextEntry will invalid the previous reader
func (s *segment) NextEntry() io.Reader {
	if s.closed {
		return closedReader{}
	}
	entry := &segmentEntry{}
	_, err := entry.ReadFrom(s.input)
	if err != nil {
		return failedReader{}
	}
	return &entryReader{entry.buf}
}

// Close the current segment releasing any resources associated with it
func (s *segment) Close() error {
	if s.readonly {
		f := s.file
		s.file = nil
		s.input = nil
		return f.Close()
	}
	err := s.output.Close()
	if err != nil {
		return err
	}

	err = s.file.Sync()
	if err != nil {
		return err
	}

	s.output = nil
	file := s.file
	s.file = nil
	return file.Close()
}

// Read implements io.Reader
func (c closedReader) Read(_ []byte) (int, error) { return 0, ErrClosed }

func (u updateSum) Write(buf []byte) (int, error) {
	u.sum.Write(buf)
	return u.out.Write(buf)
}

func (f failedReader) Read(_ []byte) (int, error) {
	return 0, f.err
}

func (e *entryReader) Read(dest []byte) (int, error) {
	n := copy(dest, e.buf)
	if n == 0 {
		return 0, io.EOF
	}
	e.buf = e.buf[n:]
	return n, nil
}
