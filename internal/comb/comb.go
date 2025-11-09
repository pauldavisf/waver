package comb

import (
	"context"
	"elon/waver/internal/pkg/wav"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type CombInfo struct {
	Filename   string
	SampleName string
	AddEmpty   bool
}

func CombineWavFiles(ctx context.Context, combInfos []CombInfo, outPath string) error {
	if len(combInfos) == 0 {
		return nil
	}

	var combinedData []byte
	var firstHeader *wav.WavHeader

	for i, info := range combInfos {
		f, err := os.Open(info.Filename)
		if err != nil {
			return fmt.Errorf("open file %s: %w", info.Filename, err)
		}
		defer f.Close()

		header, err := wav.ReadWavHeader(f)
		if err != nil {
			return fmt.Errorf("read header from %s: %w", info.Filename, err)
		}

		if i == 0 {
			firstHeader = header
		} else {
			if header.NumChannels != firstHeader.NumChannels ||
				header.SampleRate != firstHeader.SampleRate ||
				header.BitsPerSample != firstHeader.BitsPerSample {

				return fmt.Errorf("incompatible WAV files: %s and %s", combInfos[0].Filename, info.Filename)
			}
		}

		data := make([]byte, header.Subchunk2Size)
		if info.AddEmpty {
			combinedData = append(combinedData, data...)
			continue
		}

		_, err = io.ReadFull(f, data)
		if err != nil {
			return fmt.Errorf("read audio data from %s: %w", info.Filename, err)
		}
		combinedData = append(combinedData, data...)
	}

	firstHeader.Subchunk2Size = uint32(len(combinedData))
	firstHeader.ChunkSize = 36 + firstHeader.Subchunk2Size

	outFileName := filepath.Join(outPath, combInfos[0].SampleName+wav.WavExt)

	out, err := os.Create(outFileName)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	err = binary.Write(out, binary.LittleEndian, firstHeader)
	if err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	_, err = out.Write(combinedData)
	if err != nil {
		return fmt.Errorf("write audio data: %w", err)
	}

	return nil
}
