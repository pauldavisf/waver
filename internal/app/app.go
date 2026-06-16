package app

import (
	"context"
	"elon/waver/internal/comb"
	"elon/waver/internal/finder"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type CombineResult struct {
	InputPath     string `json:"inputPath"`
	OutputPath    string `json:"outputPath"`
	TrackCount    int    `json:"trackCount"`
	SegmentCount  int    `json:"segmentCount"`
	SourceCount   int    `json:"sourceCount"`
	SilenceCount  int    `json:"silenceCount"`
	OutputRoot    string `json:"outputRoot"`
	ProjectFolder string `json:"projectFolder"`
}

func Combine(ctx context.Context, base string, outputRoot string) (*CombineResult, error) {
	exists, err := isDirExists(base)
	if err != nil {
		return nil, fmt.Errorf("check dir exists: %w", err)
	}

	if !exists {
		return nil, fmt.Errorf("dir %s does not exist or is not a directory", base)
	}

	if outputRoot == "" {
		outputRoot = "out"
	}

	cleanPath := filepath.Clean(base)
	lastDir := filepath.Base(cleanPath)
	outPath := filepath.Join(outputRoot, lastDir)

	if err := ensureDir(outPath); err != nil {
		return nil, err
	}

	files, err := finder.FindFilesForPads(ctx, base, []string{})
	if err != nil {
		return nil, fmt.Errorf("find files for pads: %w", err)
	}

	pads := make([]string, 0, len(files))
	for pad := range files {
		pads = append(pads, pad)
	}
	sort.Strings(pads)

	result := &CombineResult{
		InputPath:     cleanPath,
		OutputPath:    absPath(outPath),
		OutputRoot:    outputRoot,
		ProjectFolder: lastDir,
		TrackCount:    len(files),
	}

	for _, pad := range pads {
		infos := files[pad]
		if err := comb.CombineWavFiles(ctx, infos, outPath); err != nil {
			return nil, fmt.Errorf("combine wav files for pad %s: %w", pad, err)
		}

		for _, info := range infos {
			result.SegmentCount++
			if info.AddEmpty {
				result.SilenceCount++
			} else {
				result.SourceCount++
			}
		}
	}

	return result, nil
}

func isDirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return info.IsDir(), nil
}

func ensureDir(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return fmt.Errorf("make dir: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("check dir: %w", err)
	} else if !info.IsDir() {
		return fmt.Errorf("is not a directory: %s", path)
	}

	return nil
}

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}

	return abs
}
