package trail

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestWrite(t *testing.T) {
	dir, err := ioutil.TempDir("", "trail-test")
	defer os.RemoveAll(dir)
	mustNot(t, "fail to create temp dir", err)
	log, err := New(dir, 0644)
	mustNot(t, "fail to create log", err)

	expectedEntries := [][]byte{
		[]byte("hello world"),
		[]byte("ola mundo"),
		[]byte("halo Welt"),
		[]byte("hola mundo"),
	}

	mustNot(t, "fail to append", log.Append(expectedEntries[0]))
	mustNot(t, "fail to append", log.Append(expectedEntries[1]))
	mustNot(t, "fail to pack", log.Pack())
	mustNot(t, "fail to append", log.Append(expectedEntries[2]))
	mustNot(t, "fail to append", log.Append(expectedEntries[3]))
	mustNot(t, "fail to pack", log.Pack())
	mustNot(t, "fail to close", log.Close())

	log, err = New(dir, 0644)
	mustNot(t, "fail to open a trail in the same folder as a closed trail", err)

	segmentNames, err := log.SegmentNames()
	mustNot(t, "fail to get list of segment names", err)
	if len(segmentNames) != 2 {
		t.Fatalf("Since we called Pack then Close, it should have 2 segments but got %v", len(segmentNames))
	}

	checkSegment := func(log *Trail, name string, expectedEntries [][]byte) {
		segment, err := log.OpenSegment(segmentNames[0])
		mustNot(t, "fail to open segment name", err)
		defer segment.Close()

		checkEntry := func(segment Segment, expectedValue []byte) {
			buf := &bytes.Buffer{}
			n, err := io.Copy(buf, segment.NextEntry())
			mustNot(t, "fail to copy entry to buffer", err)
			if int(n) != len(expectedValue) {
				t.Fatalf("Entry from segment should have %v bytes got got %v", len(expectedValue), n)
			}
			if !reflect.DeepEqual(buf.Bytes(), expectedValue) {
				t.Fatalf("Expecting %v but got %v", string(expectedValue), buf.String())
			}
		}

		for _, v := range expectedEntries {
			checkEntry(segment, v)
		}
	}

	checkSegment(log, segmentNames[0], expectedEntries[:2])
	checkSegment(log, segmentNames[1], expectedEntries[2:])

	mustNot(t, "fail to close", log.Close())
}
