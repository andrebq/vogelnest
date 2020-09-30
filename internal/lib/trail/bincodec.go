package trail

import (
	"encoding/binary"
	"io"
)

type (
	binWrite struct {
		w io.Writer
	}

	binReader struct {
		r interface {
			io.Reader
			io.ByteReader
		}
		err error
	}
)

func (b binWrite) Write(buf []byte) (int, error) {
	return b.w.Write(buf)
}

func (b binWrite) WriteString(str string) (int, error) {
	return io.WriteString(b.w, str)
}

func (b binWrite) WriteVarint(val int64) (int, error) {
	var aux [binary.MaxVarintLen64]byte
	n := binary.PutVarint(aux[:], val)
	return b.Write(aux[:n])
}

func (b binReader) ReadVarint() (int64, error) {
	if b.err != nil {
		return 0, b.err
	}
	var v int64
	v, b.err = binary.ReadVarint(b.r)
	return v, b.err
}

func (b binReader) Read(buf []byte) (int, error) {
	return b.r.Read(buf)
}
