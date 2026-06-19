// Package utils provides shared utility functions and abstractions used across
// the ork codebase, including filesystem operations, terminal detection, shell
// helpers, and validation.
package utils

import (
	"io"
	"os"

	"github.com/pkg/sftp"
)

// FileWriter combines io.Writer and io.Closer for writing to files on any
// filesystem implementation.
type FileWriter interface {
	io.Writer
	io.Closer
}

// FileReader combines io.Reader and io.Closer for reading files on any
// filesystem implementation.
type FileReader interface {
	io.Reader
	io.Closer
}

type FileInfo interface {
	Name() string
	IsDir() bool
	Size() int64
	Mode() os.FileMode
}

// FS is a filesystem abstraction that allows ork to operate transparently on
// local and remote (SFTP) filesystems.
type FS interface {
	Stat(path string) (os.FileInfo, error)
	IsDir(path string) (bool, error)
	Open(path string) (FileReader, error)
	ReadDir(path string) ([]FileInfo, error)
	MkdirAll(path string) error
	Create(path string) (FileWriter, error)
}

// LocalFS implements FS interface for local filesystem
type LocalFS struct{}

func (l *LocalFS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (l *LocalFS) ReadDir(path string) ([]FileInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	infos := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		fi, err := e.Info()
		if err != nil {
			return nil, err
		}
		infos = append(infos, fi)
	}
	return infos, nil
}

func (l *LocalFS) IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func (l *LocalFS) Open(path string) (FileReader, error) {
	return os.Open(path)
}

func (l *LocalFS) Create(path string) (FileWriter, error) {
	return os.Create(path)
}

func (l *LocalFS) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

// SFTPFS implements FS interface for SFTP filesystem
type SFTPFS struct {
	SftpClient *sftp.Client
}

func (s *SFTPFS) Stat(path string) (os.FileInfo, error) {
	return s.SftpClient.Stat(path)
}

func (s *SFTPFS) IsDir(path string) (bool, error) {
	info, err := s.SftpClient.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func (s *SFTPFS) Open(path string) (FileReader, error) {
	remoteFile, err := s.SftpClient.Open(path)
	if err != nil {
		return nil, err
	}

	return remoteFile, nil
}

func (s *SFTPFS) ReadDir(path string) ([]FileInfo, error) {
	entries, err := s.SftpClient.ReadDir(path)
	if err != nil {
		return nil, err
	}
	infos := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		infos = append(infos, e)
	}
	return infos, nil
}

func (s *SFTPFS) Create(path string) (FileWriter, error) {
	return s.SftpClient.Create(path)
}

func (s *SFTPFS) MkdirAll(path string) error {
	return s.SftpClient.MkdirAll(path)
}

// FSWithPath pairs a filesystem implementation with a base path, used as a
// source or destination in file copy operations.
type FSWithPath struct {
	FS   FS
	Path string
}
