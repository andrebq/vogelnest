package trail

import (
	"encoding/binary"
	"io"
)

type (
	binWrite struct {
		w io.Writer
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
