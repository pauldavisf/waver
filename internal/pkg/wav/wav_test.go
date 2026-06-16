package wav

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestReadWavHeaderFindsDataChunkAfterExtraChunks(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	source := buildTestWav(t, []testChunk{
		{id: "JUNK", data: []byte{9, 9, 9}},
	}, data)

	r := bytes.NewReader(source)
	header, err := ReadWavHeader(r)
	if err != nil {
		t.Fatalf("ReadWavHeader() error = %v", err)
	}

	if header.Subchunk2Size != uint32(len(data)) {
		t.Fatalf("Subchunk2Size = %d, want %d", header.Subchunk2Size, len(data))
	}

	got := make([]byte, header.Subchunk2Size)
	if _, err := io.ReadFull(r, got); err != nil {
		t.Fatalf("read data after header: %v", err)
	}

	if !bytes.Equal(got, data) {
		t.Fatalf("data after header = %v, want %v", got, data)
	}
}

type testChunk struct {
	id   string
	data []byte
}

func buildTestWav(t *testing.T, extraChunks []testChunk, data []byte) []byte {
	t.Helper()

	var body bytes.Buffer
	writeTestChunk(t, &body, "fmt ", testFmtChunk(t))
	for _, chunk := range extraChunks {
		writeTestChunk(t, &body, chunk.id, chunk.data)
	}
	writeTestChunk(t, &body, "data", data)

	var out bytes.Buffer
	out.WriteString("RIFF")
	writeLE(t, &out, uint32(4+body.Len()))
	out.WriteString("WAVE")
	out.Write(body.Bytes())

	return out.Bytes()
}

func testFmtChunk(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	writeLE(t, &buf, uint16(1))
	writeLE(t, &buf, uint16(1))
	writeLE(t, &buf, uint32(44100))
	writeLE(t, &buf, uint32(88200))
	writeLE(t, &buf, uint16(2))
	writeLE(t, &buf, uint16(16))

	return buf.Bytes()
}

func writeTestChunk(t *testing.T, buf *bytes.Buffer, id string, data []byte) {
	t.Helper()

	buf.WriteString(id)
	writeLE(t, buf, uint32(len(data)))
	buf.Write(data)

	if len(data)%2 == 1 {
		buf.WriteByte(0)
	}
}

func writeLE(t *testing.T, w io.Writer, data interface{}) {
	t.Helper()

	if err := binary.Write(w, binary.LittleEndian, data); err != nil {
		t.Fatalf("binary.Write() error = %v", err)
	}
}
