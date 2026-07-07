package comb

import (
	"context"
	"elon/waver/internal/pkg/wav"
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
		header, data, err := readCombData(info)
		if err != nil {
			return err
		}

		if err := validateHeader(header, info.Filename); err != nil {
			return err
		}

		if i == 0 {
			firstHeader = header
		} else {
			if !isCompatible(header, firstHeader) {
				return fmt.Errorf("incompatible WAV files: %s and %s", combInfos[0].Filename, info.Filename)
			}
		}

		combinedData = append(combinedData, data...)
	}

	dataSize := uint64(len(combinedData))
	if dataSize > uint64(^uint32(0))-36 {
		return fmt.Errorf("combined WAV data is too large: %d bytes", dataSize)
	}

	outHeader := wav.CanonicalHeader(firstHeader, uint32(dataSize))

	outFileName := filepath.Join(outPath, combInfos[0].SampleName+wav.WavExt)

	out, err := os.Create(outFileName)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	if err := wav.WriteWavHeader(out, &outHeader); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	if _, err := out.Write(combinedData); err != nil {
		return fmt.Errorf("write audio data: %w", err)
	}

	return nil
}

func readCombData(info CombInfo) (*wav.WavHeader, []byte, error) {
	f, err := os.Open(info.Filename)
	if err != nil {
		return nil, nil, fmt.Errorf("open file %s: %w", info.Filename, err)
	}
	defer f.Close()

	header, err := wav.ReadWavHeader(f)
	if err != nil {
		return nil, nil, fmt.Errorf("read header from %s: %w", info.Filename, err)
	}

	data := make([]byte, header.Subchunk2Size)
	if info.AddEmpty {
		return header, data, nil
	}

	if _, err := io.ReadFull(f, data); err != nil {
		return nil, nil, fmt.Errorf("read audio data from %s: %w", info.Filename, err)
	}

	return header, data, nil
}

func validateHeader(header *wav.WavHeader, filename string) error {
	if header.BlockAlign == 0 {
		return fmt.Errorf("invalid WAV block align in %s", filename)
	}

	if header.Subchunk2Size%uint32(header.BlockAlign) != 0 {
		return fmt.Errorf("WAV data size is not aligned to sample frames in %s", filename)
	}

	return nil
}

func isCompatible(header, firstHeader *wav.WavHeader) bool {
	return header.AudioFormat == firstHeader.AudioFormat &&
		header.NumChannels == firstHeader.NumChannels &&
		header.SampleRate == firstHeader.SampleRate &&
		header.ByteRate == firstHeader.ByteRate &&
		header.BlockAlign == firstHeader.BlockAlign &&
		header.BitsPerSample == firstHeader.BitsPerSample
}
