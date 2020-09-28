package trail

import "errors"

var (
	//ErrClosed indicates that either a segment or a log was closed and is invalid now
	ErrClosed = errors.New("closed")
)
