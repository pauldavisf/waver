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

func FindFilesForPads(ctx context.Context, basePath string, pads []string) (map[string][]comb.CombInfo, error) {
	result := make(map[string][]comb.CombInfo)
	seen := make(map[string]struct{})

	for {
		pad, err := findNotSeenPad(basePath, pads, seen)
		if err != nil {
			return nil, fmt.Errorf("find not seen pad: %w", err)
		}
		if pad == "" {
			break
		}

		infos, err := getInfosForPad(ctx, basePath, pads, pad)
		if err != nil {
			return nil, fmt.Errorf("get files for pad %s: %w", pad, err)
		}

		result[pad] = infos
		seen[pad] = struct{}{}
	}

	return result, nil
}

func getInfosForPad(ctx context.Context, basePath string, pads []string, pad string) ([]comb.CombInfo, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		panic(err)
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

			var randomFile os.DirEntry
			added := false

			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), wav.WavExt) {
					randomFile = f

					if strings.HasPrefix(f.Name(), pad) {
						fullPath := filepath.Join(subdir, f.Name())
						infos = append(infos, comb.CombInfo{
							Filename: fullPath,
							AddEmpty: false,
						})

						added = true

						break
					}
				}
			}

			if !added {
				infos = append(infos, comb.CombInfo{
					Filename: filepath.Join(subdir, randomFile.Name()),
					AddEmpty: true,
				})
			}
		}
	}

	return infos, nil
}

func findNotSeenPad(basePath string, pads []string, seen map[string]struct{}) (string, error) {
	result := ""

	entries, err := os.ReadDir(basePath)
	if err != nil {
		panic(err)
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
				return "", fmt.Errorf("read directory %s: %w", subdir, err)
			}

			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), wav.WavExt) {
					padName, err := getPadName(f.Name())
					if err != nil {
						return "", fmt.Errorf("get pad name from %s: %w", f.Name(), err)
					}

					_, ok := seen[padName]
					if !ok {
						result = padName

						break
					}
				}
			}
		}
	}

	return result, nil
}

func getPadName(source string) (string, error) {
	parts := strings.SplitN(source, "-", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid pad name format: %s", source)
	}

	return parts[0] + "-" + parts[1], nil
}
