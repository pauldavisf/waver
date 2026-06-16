package finder

import (
	"bytes"
	"context"
	"elon/waver/internal/pkg/wav"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindFilesForPadsDoesNotMatchPadNameByPrefix(t *testing.T) {
	base := t.TempDir()
	mkdirAll(t, filepath.Join(base, "pattern-01"))
	mkdirAll(t, filepath.Join(base, "pattern-02"))
	writeTinyWav(t, filepath.Join(base, "pattern-01", "C-2-kick.wav"), 4)
	writeTinyWav(t, filepath.Join(base, "pattern-02", "C-20-bass.wav"), 4)

	files, err := FindFilesForPads(context.Background(), base, nil)
	if err != nil {
		t.Fatalf("FindFilesForPads() error = %v", err)
	}

	infos := files["C-2"]
	if len(infos) != 2 {
		t.Fatalf("len(files[\"C-2\"]) = %d, want 2", len(infos))
	}

	if infos[0].AddEmpty {
		t.Fatalf("first C-2 segment is empty, want source audio")
	}

	if !infos[1].AddEmpty {
		t.Fatalf("second C-2 segment is source audio, want silence")
	}
}

func TestFindFilesForPadsUsesLongestWavAsSilenceReference(t *testing.T) {
	base := t.TempDir()
	mkdirAll(t, filepath.Join(base, "pattern-01"))
	mkdirAll(t, filepath.Join(base, "pattern-02"))
	writeTinyWav(t, filepath.Join(base, "pattern-01", "C-1-main.wav"), 4)
	writeTinyWav(t, filepath.Join(base, "pattern-02", "B-1-long.wav"), 8)
	writeTinyWav(t, filepath.Join(base, "pattern-02", "Z-9-short.wav"), 4)

	files, err := FindFilesForPads(context.Background(), base, nil)
	if err != nil {
		t.Fatalf("FindFilesForPads() error = %v", err)
	}

	infos := files["C-1"]
	if len(infos) != 2 {
		t.Fatalf("len(files[\"C-1\"]) = %d, want 2", len(infos))
	}

	if !infos[1].AddEmpty {
		t.Fatalf("second C-1 segment is source audio, want silence")
	}

	if got := filepath.Base(infos[1].Filename); got != "B-1-long.wav" {
		t.Fatalf("silence reference = %s, want B-1-long.wav", got)
	}
}

func TestFindFilesForPadsReturnsErrorForPatternWithoutWav(t *testing.T) {
	base := t.TempDir()
	mkdirAll(t, filepath.Join(base, "pattern-01"))
	mkdirAll(t, filepath.Join(base, "pattern-02"))
	writeTinyWav(t, filepath.Join(base, "pattern-01", "C-1-main.wav"), 4)
	writeFile(t, filepath.Join(base, "pattern-02", "notes.txt"), []byte("no wav here"))

	_, err := FindFilesForPads(context.Background(), base, nil)
	if err == nil {
		t.Fatalf("FindFilesForPads() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "no WAV files") {
		t.Fatalf("FindFilesForPads() error = %v, want no WAV files error", err)
	}
}

func writeTinyWav(t *testing.T, filename string, dataSize int) {
	t.Helper()

	if dataSize%2 != 0 {
		t.Fatalf("test WAV data size must align to 16-bit mono frames")
	}

	baseHeader := wav.WavHeader{
		AudioFormat:   1,
		NumChannels:   1,
		SampleRate:    44100,
		ByteRate:      88200,
		BlockAlign:    2,
		BitsPerSample: 16,
	}
	header := wav.CanonicalHeader(&baseHeader, uint32(dataSize))

	var buf bytes.Buffer
	if err := wav.WriteWavHeader(&buf, &header); err != nil {
		t.Fatalf("write WAV header: %v", err)
	}
	buf.Write(make([]byte, dataSize))

	writeFile(t, filename, buf.Bytes())
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
}

func writeFile(t *testing.T, filename string, data []byte) {
	t.Helper()

	if err := os.WriteFile(filename, data, 0644); err != nil {
		t.Fatalf("WriteFile(%q): %v", filename, err)
	}
}
