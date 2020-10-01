package storage

import (
	"bytes"
	"io"

	"github.com/andrebq/vogelnest/internal/lib/trail"
	"github.com/andrebq/vogelnest/internal/schema"
	"google.golang.org/protobuf/encoding/protojson"
)

type (
	TweetLogReader struct {
		trail.Segment
	}
)

// OpenLog returns a TweetLogReader that operates on top of a Trail segment
func OpenLog(filename string) (*TweetLogReader, error) {
	seg, err := trail.OpenSegment(filename)
	if err != nil {
		return nil, err
	}
	return &TweetLogReader{
		Segment: seg,
	}, nil
}

// Next returns the next tweet entry
func (t *TweetLogReader) Next() (*schema.Tweet, error) {
	buf := bytes.Buffer{}
	_, err := io.Copy(&buf, t.NextEntry())
	if err != nil {
		return nil, err
	}
	var entry schema.Tweet
	err = protojson.Unmarshal(buf.Bytes(), &entry)
	return &entry, err
}
