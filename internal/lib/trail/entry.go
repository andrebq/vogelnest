package trail

import (
	"io"
)

type (
	segmentEntry struct {
		buf []byte
	}

	accWriter struct {
		w            io.Writer
		err          error
		totalWritten int64
	}
)

func (s segmentEntry) WriteTo(o io.Writer) (int64, error) {
	ac := &accWriter{w: o}
	bw := binWrite{ac}
	bw.WriteVarint(int64(len(s.buf)))
	bw.Write(s.buf)
	bw.WriteVarint(int64(len(s.buf)))
	return ac.totalWritten, ac.err
}

func (a *accWriter) Write(buf []byte) (int, error) {
	if a.err != nil {
		return 0, a.err
	}
	n, err := a.w.Write(buf)
	a.err = err
	a.totalWritten += int64(n)
	return n, err
}
