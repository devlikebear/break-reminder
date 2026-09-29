package timetools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/devlikebear/break-reminder/internal/i18n"
	"os"
	"path/filepath"
	"syscall"
)

type Store struct{ Path string }

func DefaultDirectory() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "break-reminder")
}
func NewStore(dir string) *Store { return &Store{Path: filepath.Join(dir, "time-tools.json")} }
func (s *Store) Load() (Snapshot, error) {
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return NewSnapshot(), nil
	}
	if err != nil {
		return Snapshot{}, err
	}
	var snapshot Snapshot
	if err = json.Unmarshal(data, &snapshot); err != nil {
		return snapshot, fail("store_corrupt", i18n.Text("Cannot read time-tools data. Preserve the original file and repair it"))
	}
	return snapshot, snapshot.Validate()
}
func (s *Store) Update(revision *uint64, fn func(Snapshot) (Snapshot, error)) (Snapshot, error) {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0700); err != nil {
		return Snapshot{}, err
	}
	lock, err := os.OpenFile(s.Path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return Snapshot{}, err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return Snapshot{}, err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	previous, err := s.Load()
	if err != nil {
		return previous, err
	}
	if revision != nil && *revision != previous.Revision {
		return previous, fail("conflict", i18n.Text("Another window changed the state. Refresh and try again"))
	}
	next, err := fn(previous.clone())
	if err != nil {
		return previous, err
	}
	if err = next.Validate(); err != nil {
		return previous, err
	}
	before, _ := json.Marshal(previous)
	after, err := json.Marshal(next)
	if err != nil {
		return previous, err
	}
	if bytes.Equal(before, after) {
		return previous, nil
	}
	if previous.Revision == ^uint64(0) {
		return previous, fmt.Errorf("%s", i18n.Text("revision overflow"))
	}
	next.Revision = previous.Revision + 1
	data, err := json.Marshal(next)
	if err != nil {
		return previous, err
	}
	if err = atomicWrite(s.Path, data); err != nil {
		return previous, err
	}
	return next, nil
}
func atomicWrite(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
