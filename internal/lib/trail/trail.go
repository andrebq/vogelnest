package trail

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type (
	// Trail manages a series of log files inside a given directory
	Trail struct {
		dir           string
		closed        bool
		err           error
		mode          os.FileMode
		activeSegment *segment
	}

	// Segment expose a read-only segment as any write operation
	// MUST be executed by the Trail object itself
	Segment interface {
		io.Closer
		NextEntry() io.Reader
	}
)

var (
	rePackedSegment = regexp.MustCompile(`\d+_\d+_[\d|a-z]+_[\d|a-z]+.segment.gz`)
)

// New trail saving files in the given directory
func New(dir string, segmentFileMode os.FileMode) (*Trail, error) {
	t := &Trail{
		dir: dir,
	}
	t.err = t.startSegment()
	if t.err != nil {
		return nil, t.err
	}
	return t, nil
}

// Append the given content to the active segment
func (t *Trail) Append(content []byte) error {
	if t.closed {
		return ErrClosed
	}
	_, err := t.activeSegment.append(segmentEntry{buf: content})
	return err
}

// Pack the current segment and open a new one
func (t *Trail) Pack() error {
	if t.err != nil {
		return t.err
	}
	t.err = t.activeSegment.packAndClose()
	if t.err != nil {
		return t.err
	}
	t.err = t.startSegment()
	return t.err
}

// Close the current log (the current segment is packed)
func (t *Trail) Close() error {
	if t.closed {
		return nil
	}
	t.closed = true
	err := t.activeSegment.packAndClose()
	t.activeSegment = nil
	return err
}

// Size returns the total size of this log,
// including packed entries and the active log,
// the active log might change as the file is not synced
// on every write
func (t *Trail) Size() (int64, error) {
	segments, err := t.SegmentNames()
	if err != nil {
		return 0, err
	}
	segments = append(segments, filepath.Base(t.activeSegment.file.Name()))
	total := int64(0)
	for _, s := range segments {
		info, err := os.Lstat(filepath.Join(t.dir, s))
		if err != nil {
			return 0, err
		}
		total += info.Size()
	}
	return total, nil
}

// SegmentNames returns the list of segments that this Trail
// has access to
func (t *Trail) SegmentNames() ([]string, error) {
	if t.closed {
		return nil, ErrClosed
	}
	var ret []string
	err := filepath.Walk(t.dir, func(path string, info os.FileInfo, err error) error {
		name := filepath.Base(info.Name())
		if rePackedSegment.MatchString(name) {
			ret = append(ret, name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// OpenSegment looks up and returns the segment with the
// given name or an error if the segment is not available
func (t *Trail) OpenSegment(name string) (Segment, error) {
	segfile := filepath.Join(t.dir, filepath.Clean(name))
	file, err := os.Open(segfile)
	if err != nil {
		return nil, err
	}
	gzr, err := gzip.NewReader(file)
	if err != nil {
		file.Close()
		return nil, err
	}
	return &segment{
		readonly: true,
		file:     file,
		input:    bufio.NewReader(gzr),
	}, nil
}

func (t *Trail) startSegment() error {
	seg := &segment{}
	fd, err := os.OpenFile(filepath.Join(t.dir, "active.gz"), os.O_CREATE|os.O_EXCL, t.mode)
	if err != nil {
		return err
	}
	gz, err := gzip.NewWriterLevel(fd, gzip.BestCompression)
	if err != nil {
		return err
	}
	seg.output = gz
	seg.file = fd
	t.activeSegment = seg
	return nil
}
