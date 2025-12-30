package utils

import (
	"fmt"
	"io"
	"path"
	"time"
)

type FSCopyResult struct {
	TotalBytes int64
	TotalFiles int
	Duration   time.Duration
}

type FSCopyOptions struct {
	Recursive bool
	Overwrite bool
}

func FSCopy(from, to FSWithPath, opts FSCopyOptions) (FSCopyResult, error) {

	totalBytes := int64(0)
	totalFiles := 0
	start := time.Now()

	var walk func(from, to FSWithPath) error
	walk = func(from, to FSWithPath) error {
		fromPath := from.Path
		toPath := to.Path
		info, err := from.FS.Stat(fromPath)
		if err != nil {
			return err
		}

		if info.IsDir() {
			if !opts.Recursive {
				return fmt.Errorf("%s is a directory, recursive=false", fromPath)
			}
			entries, err := from.FS.ReadDir(fromPath)
			if err != nil {
				return err
			}
			for _, e := range entries {
				eFrom := FSWithPath{FS: from.FS, Path: path.Join(fromPath, e.Name())}
				eTo := FSWithPath{FS: to.FS, Path: path.Join(to.Path, e.Name())}
				if err := walk(eFrom, eTo); err != nil {
					return err
				}
			}
			return nil
		}

		// Copy file
		srcFile, err := from.FS.Open(fromPath)
		if err != nil {
			return err
		}
		defer func(srcFile FileReader) {
			err := srcFile.Close()
			if err != nil {
			}
		}(srcFile)

		if !opts.Overwrite {
			if _, err := to.FS.Stat(toPath); err == nil {
				return fmt.Errorf("file %s already exists", toPath)
			}
		}

		if err := to.FS.MkdirAll(path.Dir(toPath)); err != nil {
			return err
		}
		dstFile, err := to.FS.Create(toPath)
		if err != nil {
			return err
		}
		defer func(dstFile FileWriter) {
			err := dstFile.Close()
			if err != nil {

			}
		}(dstFile)

		n, err := io.Copy(dstFile, srcFile)
		if err != nil {
			return err
		}
		totalBytes += n
		totalFiles++
		return nil
	}

	err := walk(from, to)

	return FSCopyResult{
		TotalBytes: totalBytes,
		TotalFiles: totalFiles,
		Duration:   time.Since(start),
	}, err
}
