package comb

import (
	"bytes"
	"context"
	"elon/waver/internal/pkg/wav"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCombineWavFilesReadsDataAfterExtraChunks(t *testing.T) {
	inDir := t.TempDir()
	outDir := t.TempDir()
	first := filepath.Join(inDir, "C-2-first.wav")
	second := filepath.Join(inDir, "C-2-second.wav")
	writeCombTestWav(t, first, []byte{1, 2, 3, 4})
	writeCombTestWav(t, second, []byte{5, 6, 7, 8})

	err := CombineWavFiles(context.Background(), []CombInfo{
		{Filename: first, SampleName: "C-2"},
		{Filename: second, SampleName: "C-2"},
	}, outDir)
	if err != nil {
		t.Fatalf("CombineWavFiles() error = %v", err)
	}

	got := readWavData(t, filepath.Join(outDir, "C-2.wav"))
	want := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	if !bytes.Equal(got, want) {
		t.Fatalf("combined data = %v, want %v", got, want)
	}
}

func TestCombineWavFilesWritesSilenceForEmptySegment(t *testing.T) {
	inDir := t.TempDir()
	outDir := t.TempDir()
	first := filepath.Join(inDir, "C-2-first.wav")
	second := filepath.Join(inDir, "C-2-empty-reference.wav")
	writeCombTestWav(t, first, []byte{1, 2, 3, 4})
	writeCombTestWav(t, second, []byte{9, 9, 9, 9})

	err := CombineWavFiles(context.Background(), []CombInfo{
		{Filename: first, SampleName: "C-2"},
		{Filename: second, SampleName: "C-2", AddEmpty: true},
	}, outDir)
	if err != nil {
		t.Fatalf("CombineWavFiles() error = %v", err)
	}

	got := readWavData(t, filepath.Join(outDir, "C-2.wav"))
	want := []byte{1, 2, 3, 4, 0, 0, 0, 0}
	if !bytes.Equal(got, want) {
		t.Fatalf("combined data = %v, want %v", got, want)
	}
}

func writeCombTestWav(t *testing.T, filename string, data []byte) {
	t.Helper()

	var body bytes.Buffer
	writeCombChunk(t, &body, "fmt ", combFmtChunk(t))
	writeCombChunk(t, &body, "JUNK", []byte{7, 7, 7})
	writeCombChunk(t, &body, "data", data)

	var out bytes.Buffer
	out.WriteString("RIFF")
	writeCombLE(t, &out, uint32(4+body.Len()))
	out.WriteString("WAVE")
	out.Write(body.Bytes())

	if err := os.WriteFile(filename, out.Bytes(), 0644); err != nil {
		t.Fatalf("WriteFile(%q): %v", filename, err)
	}
}

func readWavData(t *testing.T, filename string) []byte {
	t.Helper()

	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("Open(%q): %v", filename, err)
	}
	defer f.Close()

	header, err := wav.ReadWavHeader(f)
	if err != nil {
		t.Fatalf("ReadWavHeader(%q): %v", filename, err)
	}

	data := make([]byte, header.Subchunk2Size)
	if _, err := io.ReadFull(f, data); err != nil {
		t.Fatalf("ReadFull(%q): %v", filename, err)
	}

	return data
}

func combFmtChunk(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	writeCombLE(t, &buf, uint16(1))
	writeCombLE(t, &buf, uint16(1))
	writeCombLE(t, &buf, uint32(44100))
	writeCombLE(t, &buf, uint32(88200))
	writeCombLE(t, &buf, uint16(2))
	writeCombLE(t, &buf, uint16(16))

	return buf.Bytes()
}

func writeCombChunk(t *testing.T, buf *bytes.Buffer, id string, data []byte) {
	t.Helper()

	buf.WriteString(id)
	writeCombLE(t, buf, uint32(len(data)))
	buf.Write(data)

	if len(data)%2 == 1 {
		buf.WriteByte(0)
	}
}

func writeCombLE(t *testing.T, w io.Writer, data interface{}) {
	t.Helper()

	if err := binary.Write(w, binary.LittleEndian, data); err != nil {
		t.Fatalf("binary.Write() error = %v", err)
	}
}
