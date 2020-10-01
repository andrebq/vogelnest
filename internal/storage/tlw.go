package storage

import (
	"github.com/andrebq/vogelnest/internal/lib/trail"
	"github.com/andrebq/vogelnest/internal/schema"
	"google.golang.org/protobuf/encoding/protojson"
)

type (
	TweetLogWriter struct {
		*trail.Trail
	}
)

// NewTweetLogWriter returns a writer which will create a
// trail.Trail object inside the given directory
func NewLog(dir string) (*TweetLogWriter, error) {
	tlw := &TweetLogWriter{}
	var err error
	tlw.Trail, err = trail.New(dir, 0644, true)
	if err != nil {
		return nil, err
	}
	err = tlw.Trail.Append([]byte("{}"))
	if err != nil {
		return nil, err
	}
	return tlw, nil
}

// Append the entries to the trail
func (t *TweetLogWriter) Append(entries ...*schema.Tweet) error {
	for _, e := range entries {
		buf, err := protojson.Marshal(e)
		if err != nil {
			return err
		}
		err = t.Trail.Append(buf)
		if err != nil {
			return err
		}
	}
	return nil
}
