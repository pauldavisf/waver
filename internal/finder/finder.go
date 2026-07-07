package finder

import (
	"context"
	"elon/waver/internal/comb"
	"elon/waver/internal/pkg/wav"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/exp/slices"
)

type pad struct {
	SampleName string
	PadName    string
}

func FindFilesForPads(ctx context.Context, basePath string, pads []string) (map[string][]comb.CombInfo, error) {
	result := make(map[string][]comb.CombInfo)
	seen := make(map[string]struct{})

	for {
		pad, err := findNotSeenPad(basePath, pads, seen)
		if err != nil {
			return nil, fmt.Errorf("find not seen pad: %w", err)
		}
		if pad == nil {
			break
		}

		infos, err := getInfosForPad(ctx, basePath, pads, *pad)
		if err != nil {
			return nil, fmt.Errorf("get files for pad %v: %w", pad, err)
		}

		result[pad.PadName] = infos
		seen[pad.PadName] = struct{}{}
	}

	return result, nil
}

func getInfosForPad(ctx context.Context, basePath string, pads []string, pad pad) ([]comb.CombInfo, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	infos := make([]comb.CombInfo, 0, 10)

	for _, entry := range entries {
		if entry.IsDir() {
			if len(pads) > 0 {
				if !slices.Contains(pads, entry.Name()) {
					continue
				}
			}

			subdir := filepath.Join(basePath, entry.Name())
			files, err := os.ReadDir(subdir)
			if err != nil {
				return nil, fmt.Errorf("read directory %s: %w", subdir, err)
			}

			var silenceFile string
			var silenceSize uint32
			added := false

			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), wav.WavExt) {
					fullPath := filepath.Join(subdir, f.Name())
					dataSize, err := getWavDataSize(fullPath)
					if err != nil {
						return nil, err
					}

					if silenceFile == "" || dataSize > silenceSize {
						silenceFile = fullPath
						silenceSize = dataSize
					}

					padName, err := getPadName(f.Name())
					if err != nil {
						return nil, fmt.Errorf("get pad name from %s: %w", f.Name(), err)
					}

					if padName == pad.PadName {
						infos = append(infos, comb.CombInfo{
							Filename:   fullPath,
							SampleName: pad.SampleName,
							AddEmpty:   false,
						})

						added = true

						break
					}
				}
			}

			if !added {
				if silenceFile == "" {
					return nil, fmt.Errorf("no WAV files in pattern directory %s", subdir)
				}

				infos = append(infos, comb.CombInfo{
					Filename:   silenceFile,
					SampleName: pad.SampleName,
					AddEmpty:   true,
				})
			}
		}
	}

	return infos, nil
}

func findNotSeenPad(basePath string, pads []string, seen map[string]struct{}) (*pad, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if len(pads) > 0 {
				if !slices.Contains(pads, entry.Name()) {
					continue
				}
			}

			subdir := filepath.Join(basePath, entry.Name())
			files, err := os.ReadDir(subdir)
			if err != nil {
				return nil, fmt.Errorf("read directory %s: %w", subdir, err)
			}

			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), wav.WavExt) {
					padName, err := getPadName(f.Name())
					if err != nil {
						return nil, fmt.Errorf("get pad name from %s: %w", f.Name(), err)
					}

					_, ok := seen[padName]
					if !ok {
						return &pad{
							SampleName: strings.TrimSuffix(f.Name(), filepath.Ext(f.Name())),
							PadName:    padName,
						}, nil
					}
				}
			}
		}
	}

	return nil, nil
}

func getPadName(source string) (string, error) {
	name := strings.TrimSuffix(source, filepath.Ext(source))
	parts := strings.SplitN(name, "-", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid pad name format: %s", source)
	}

	return parts[0] + "-" + parts[1], nil
}

func getWavDataSize(filename string) (uint32, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, fmt.Errorf("open WAV file %s: %w", filename, err)
	}
	defer f.Close()

	header, err := wav.ReadWavHeader(f)
	if err != nil {
		return 0, fmt.Errorf("read WAV header from %s: %w", filename, err)
	}

	return header.Subchunk2Size, nil
}
