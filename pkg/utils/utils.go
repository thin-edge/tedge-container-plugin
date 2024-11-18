package utils

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

func PathExists(p string) bool {
	_, error := os.Stat(p)
	return !errors.Is(error, os.ErrNotExist)
}

func CopyFile(src string, dst string) error {
	// Read all content of src to data, may cause OOM for a large file.
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	// Write data to dst
	err = os.WriteFile(dst, data, 0644)
	return err
}

func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// Check if a directory is writable.
// It tries to create a dummy file in the directory to verify
func IsDirWritable(d string, perm fs.FileMode) (bool, error) {
	dirErr := os.MkdirAll(d, 0755)
	if dirErr != nil {
		return false, dirErr
	}

	file, err := os.CreateTemp(d, ".write-test")
	if err != nil {
		return false, err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	return true, nil
}

func RootDir(p string) string {
	for {
		v1 := filepath.Dir(p)
		v2 := filepath.Dir(v1)
		if v1 == v2 {
			return p
		}
		p = v1
	}
}
