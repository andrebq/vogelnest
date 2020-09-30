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

	accReader struct {
		i interface {
			io.Reader
			io.ByteReader
		}
		err       error
		totalRead int64
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

func (s *segmentEntry) ReadFrom(i interface {
	io.Reader
	io.ByteReader
}) (int64, error) {
	ac := &accReader{i: i}
	br := binReader{ac, nil}
	sz, err := br.ReadVarint()
	if err != nil {
		return ac.totalRead, err
	}
	s.buf = make([]byte, int(sz))
	_, err = br.Read(s.buf)
	if err != nil {
		return ac.totalRead, err
	}
	_, err = br.ReadVarint()
	return ac.totalRead, err
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

func (a *accReader) ReadByte() (byte, error) {
	if a.err != nil {
		return 0, a.err
	}
	var b byte
	b, a.err = a.i.ReadByte()
	if a.err == nil {
		a.totalRead++
	}
	return b, a.err
}

func (a *accReader) Read(buf []byte) (int, error) {
	if a.err != nil {
		return 0, a.err
	}
	var n int
	n, a.err = a.i.Read(buf)
	a.totalRead += int64(n)
	return n, a.err
}
