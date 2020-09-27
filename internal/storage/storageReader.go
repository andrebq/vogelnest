package storage

import (
	"bufio"
	"io"
	"os"

	"github.com/andrebq/vogelnest/internal/schema"
	"google.golang.org/protobuf/encoding/protojson"
)

type (
	// TweetLogReader is used to unpack and read
	// messages written by TweetLogWriter
	TweetLogReader struct {
		input *bufio.Scanner
		close io.Closer

		err error

		nextTag   tlvTag
		nextTweet schema.Tweet
	}
)

func OpenPackedFile(filename string) (*TweetLogReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	// reader, err := gzip.NewReader(file)
	// if err != nil {
	// 	file.Close()
	// 	return nil, err
	// }
	return &TweetLogReader{
		input: bufio.NewScanner(file),
		close: file,
	}, nil
}

func (tl *TweetLogReader) Err() error {
	if tl.err == ErrClosed {
		return nil
	}
	return tl.err
}

func (tl *TweetLogReader) Next() bool {
	if !tl.input.Scan() {
		return false
	}
	// loop while we have data to process
	for tl.err == nil {
		tl.err = protojson.Unmarshal(tl.input.Bytes(), &tl.nextTweet)
		if tl.err != nil {
			if tl.input.Scan() {
				tl.err = nil
			}
		}
	}
	tl.err = tl.input.Err()

	return tl.err == nil
}

func (tl *TweetLogReader) Entry() *schema.Tweet {
	return &tl.nextTweet
}

func (tl *TweetLogReader) Close() error {
	err := tl.close.Close()
	tl.input = nil
	tl.close = nil
	tl.err = ErrClosed
	return err
}
