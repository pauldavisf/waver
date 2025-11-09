package main

import (
	"context"
	"elon/waver/internal/comb"
	"elon/waver/internal/finder"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	args := os.Args
	if len(args) <= 1 {
		panic("no commands in arguments")
	}

	if strings.ToLower(args[1]) == "comb" {
		if len(args) <= 2 {
			panic("no pattern path in args")
		}

		err := combine(args[2])
		if err != nil {
			panic(err)
		}

		return
	}

	panic(fmt.Sprintf("unknown command: %s", args[1]))
}

func combine(base string) error {
	exists, err := isDirExists(base)
	if err != nil {
		return fmt.Errorf("check dir exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("dir %s not exists", base)
	}

	cleanPath := filepath.Clean(base)
	lastDir := filepath.Base(cleanPath)

	err = ensureDir("out/" + lastDir)
	if err != nil {
		return err
	}

	files, err := finder.FindFilesForPads(context.Background(), base, []string{})
	if err != nil {
		return fmt.Errorf("find files for pads: %w", err)
	}

	for _, infos := range files {
		err := comb.CombineWavFiles(context.Background(), infos, "out/"+lastDir+"/")
		if err != nil {
			return fmt.Errorf("combine wav files: %w", err)
		}
	}

	return nil
}

func isDirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}

	return true, err
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
