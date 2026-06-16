package wav

import (
	"encoding/binary"
	"fmt"
	"io"
)

const WavExt = ".wav"

type WavHeader struct {
	ChunkID       [4]byte
	ChunkSize     uint32
	Format        [4]byte
	Subchunk1ID   [4]byte
	Subchunk1Size uint32
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
	Subchunk2ID   [4]byte
	Subchunk2Size uint32
}

func ReadWavHeader(r io.Reader) (*WavHeader, error) {
	var header WavHeader

	if _, err := io.ReadFull(r, header.ChunkID[:]); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &header.ChunkSize); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(r, header.Format[:]); err != nil {
		return nil, err
	}

	if string(header.ChunkID[:]) != "RIFF" || string(header.Format[:]) != "WAVE" {
		return nil, fmt.Errorf("not a valid WAV file")
	}

	hasFmt := false

	for {
		var chunkID [4]byte
		if _, err := io.ReadFull(r, chunkID[:]); err != nil {
			return nil, fmt.Errorf("read WAV chunk id: %w", err)
		}

		var chunkSize uint32
		if err := binary.Read(r, binary.LittleEndian, &chunkSize); err != nil {
			return nil, fmt.Errorf("read WAV chunk size: %w", err)
		}

		switch string(chunkID[:]) {
		case "fmt ":
			if chunkSize < 16 {
				return nil, fmt.Errorf("invalid fmt chunk size: %d", chunkSize)
			}

			header.Subchunk1ID = chunkID
			header.Subchunk1Size = chunkSize

			if err := binary.Read(r, binary.LittleEndian, &header.AudioFormat); err != nil {
				return nil, fmt.Errorf("read audio format: %w", err)
			}
			if err := binary.Read(r, binary.LittleEndian, &header.NumChannels); err != nil {
				return nil, fmt.Errorf("read channel count: %w", err)
			}
			if err := binary.Read(r, binary.LittleEndian, &header.SampleRate); err != nil {
				return nil, fmt.Errorf("read sample rate: %w", err)
			}
			if err := binary.Read(r, binary.LittleEndian, &header.ByteRate); err != nil {
				return nil, fmt.Errorf("read byte rate: %w", err)
			}
			if err := binary.Read(r, binary.LittleEndian, &header.BlockAlign); err != nil {
				return nil, fmt.Errorf("read block align: %w", err)
			}
			if err := binary.Read(r, binary.LittleEndian, &header.BitsPerSample); err != nil {
				return nil, fmt.Errorf("read bits per sample: %w", err)
			}
			if err := skipChunkRest(r, chunkSize-16); err != nil {
				return nil, fmt.Errorf("skip fmt chunk extension: %w", err)
			}

			hasFmt = true

		case "data":
			if !hasFmt {
				return nil, fmt.Errorf("data chunk before fmt chunk")
			}

			header.Subchunk2ID = chunkID
			header.Subchunk2Size = chunkSize

			return &header, nil

		default:
			if err := skipChunkRest(r, chunkSize); err != nil {
				return nil, fmt.Errorf("skip %s chunk: %w", string(chunkID[:]), err)
			}
		}

		if chunkSize%2 == 1 {
			if err := skipChunkRest(r, 1); err != nil {
				return nil, fmt.Errorf("skip WAV chunk padding: %w", err)
			}
		}
	}
}

func skipChunkRest(r io.Reader, size uint32) error {
	if size == 0 {
		return nil
	}

	_, err := io.CopyN(io.Discard, r, int64(size))
	return err
}

func WriteWavHeader(w io.Writer, header *WavHeader) error {
	return binary.Write(w, binary.LittleEndian, header)
}

func CanonicalHeader(header *WavHeader, dataSize uint32) WavHeader {
	result := *header
	result.ChunkID = [4]byte{'R', 'I', 'F', 'F'}
	result.Format = [4]byte{'W', 'A', 'V', 'E'}
	result.Subchunk1ID = [4]byte{'f', 'm', 't', ' '}
	result.Subchunk1Size = 16
	result.Subchunk2ID = [4]byte{'d', 'a', 't', 'a'}
	result.Subchunk2Size = dataSize
	result.ChunkSize = 36 + result.Subchunk2Size

	return result
}
