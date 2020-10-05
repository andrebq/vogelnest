package trail

import (
	"bytes"
	"crypto/rand"
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
	log, err := New(dir, 0644, false)
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

	unpacked, err := log.UnpackedSize()
	mustNot(t, "fail to compute unpacked size", err)
	expectedUnpacked := int64(10)
	if unpacked != expectedUnpacked {
		t.Errorf("Should have only %v of unpacked bytes but got %v", expectedUnpacked, unpacked)
	}

	mustNot(t, "fail to pack", log.Pack())
	mustNot(t, "fail to close", log.Close())

	log, err = New(dir, 0644, false)
	mustNot(t, "fail to open a trail in the same folder as a closed trail", err)

	segmentNames, err := log.SegmentNames()
	mustNot(t, "fail to get list of segment names", err)
	if len(segmentNames) != 2 {
		t.Fatalf("Since we called Pack then Close, it should have 2 segments but got %v", len(segmentNames))
	}

	checkSegment := func(log *Trail, name string, expectedEntries ...[]byte) {
		segment, err := log.OpenSegment(name)
		mustNot(t, "fail to open segment name", err)
		defer segment.Close()

		checkEntry := func(segment Segment, expectedValue []byte) {
			buf := &bytes.Buffer{}
			n, err := io.Copy(buf, segment.NextEntry())
			mustNot(t, "fail to copy entry to buffer", err)
			if int(n) != len(expectedValue) {
				t.Errorf("Entry from segment should have %v bytes got got %v", len(expectedValue), n)
			}
			if !reflect.DeepEqual(buf.Bytes(), expectedValue) {
				t.Errorf("Expecting %v but got %v", string(expectedValue), buf.String())
			}
		}

		for _, v := range expectedEntries {
			checkEntry(segment, v)
		}
	}

	checkSegment(log, segmentNames[0], expectedEntries[:2]...)
	checkSegment(log, segmentNames[1], expectedEntries[2:]...)

	expectedSize := int64(95)
	size, err := log.Size()
	mustNot(t, "fail to compute the size", err)
	if size != expectedSize {
		t.Errorf("Size should be %v bytes but got %v", expectedSize, size)
	}

	expectedTrim := []string{segmentNames[0]}
	segmentsToTrim, err := log.ComputeTrim(expectedSize / 2)
	mustNot(t, "fail to compute how many segments should be trimmed", err)
	if !reflect.DeepEqual(expectedTrim, segmentsToTrim) {
		t.Errorf("Should have selected %v for trim but got %v", expectedTrim, segmentsToTrim)
	}

	err = log.Trim(segmentsToTrim...)
	mustNot(t, "fail to trim segments", err)
	segmentsToTrim, err = log.ComputeTrim(expectedSize / 2)
	mustNot(t, "fail to compute segments to trim after trim", err)
	if len(segmentsToTrim) > 0 {
		t.Errorf("After trim shouldn't have any segments to trim but got: %v", segmentsToTrim)
	}

	mustNot(t, "fail to close", log.Close())
}

func BenchmarkAppendRandom(b *testing.B) {
	b.StopTimer()
	dir, err := ioutil.TempDir("", "trail-test")
	defer os.RemoveAll(dir)
	mustNot(b, "fail to create temp dir", err)
	log, err := New(dir, 0644, false)
	mustNot(b, "fail to create log", err)

	segments := b.N
	entriesPerSegment := 100
	entrySize := 3000

	randomData := make([]byte, entrySize*entriesPerSegment*segments)
	_, err = rand.Read(randomData)
	buf := bytes.NewBuffer(randomData)
	mustNot(b, "fail to populate random data", err)
	scratchBuffer := make([]byte, entrySize)
	b.StartTimer()
	for segments > 0 {
		missingEntries := entriesPerSegment
		for missingEntries > 0 {
			buf.Read(scratchBuffer)
			err = log.Append(scratchBuffer)
			mustNot(b, "fail to write entry to log", err)
			missingEntries--
		}
		mustNot(b, "fail to pack segment", log.Pack())
		segments--
	}
}
