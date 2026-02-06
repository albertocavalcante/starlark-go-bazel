//go:build js && wasm

package main

import (
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

// MemoryFS is an in-memory filesystem for WASM.
type MemoryFS struct {
	files map[string][]byte
}

// NewMemoryFS creates a new in-memory filesystem.
func NewMemoryFS() *MemoryFS {
	return &MemoryFS{
		files: make(map[string][]byte),
	}
}

// ReadFile reads the contents of the file at path.
func (m *MemoryFS) ReadFile(path string) ([]byte, error) {
	// Normalize path
	path = normalizePath(path)
	content, ok := m.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return content, nil
}

// AddFile adds a file to the in-memory filesystem.
func (m *MemoryFS) AddFile(path string, content []byte) {
	path = normalizePath(path)
	m.files[path] = content
}

// RemoveFile removes a file from the in-memory filesystem.
func (m *MemoryFS) RemoveFile(path string) {
	path = normalizePath(path)
	delete(m.files, path)
}

// Stat returns file info for the given path.
func (m *MemoryFS) Stat(path string) (fs.FileInfo, error) {
	path = normalizePath(path)
	content, ok := m.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return &memoryFileInfo{
		name: filepath.Base(path),
		size: int64(len(content)),
	}, nil
}

// Join joins path elements.
func (m *MemoryFS) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Abs returns the absolute path.
func (m *MemoryFS) Abs(path string) (string, error) {
	return normalizePath(path), nil
}

// ListFiles returns all files in the filesystem.
func (m *MemoryFS) ListFiles() []string {
	files := make([]string, 0, len(m.files))
	for path := range m.files {
		files = append(files, path)
	}
	return files
}

// Clear removes all files from the filesystem.
func (m *MemoryFS) Clear() {
	m.files = make(map[string][]byte)
}

// normalizePath normalizes a file path.
func normalizePath(path string) string {
	// Remove leading ./
	path = strings.TrimPrefix(path, "./")
	// Clean the path
	path = filepath.Clean(path)
	// Ensure forward slashes
	path = strings.ReplaceAll(path, "\\", "/")
	return path
}

// memoryFileInfo implements fs.FileInfo for in-memory files.
type memoryFileInfo struct {
	name string
	size int64
}

func (fi *memoryFileInfo) Name() string       { return fi.name }
func (fi *memoryFileInfo) Size() int64        { return fi.size }
func (fi *memoryFileInfo) Mode() fs.FileMode  { return 0644 }
func (fi *memoryFileInfo) ModTime() time.Time { return time.Time{} }
func (fi *memoryFileInfo) IsDir() bool        { return false }
func (fi *memoryFileInfo) Sys() any           { return nil }
