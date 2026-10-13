package container_group

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
)

// projectFiles are the files which describe an installed compose project, and
// which must stay consistent with each other (the compose file defines which
// images are used, and the version file records which version that is)
var projectFiles = []string{
	"docker-compose.yaml",
	"docker-compose.yml",
	"version",
}

type fileBackup struct {
	path string
	data []byte
	mode os.FileMode
}

// ProjectBackup holds a copy of the project files of an already installed
// compose project, so that they can be restored if a later install fails
type ProjectBackup struct {
	dir   string
	files []fileBackup
}

// BackupProject reads the existing project files from dir.
// Files which do not exist are skipped, so a first time install results in an empty backup.
func BackupProject(dir string) (*ProjectBackup, error) {
	backup := &ProjectBackup{dir: dir}
	for _, name := range projectFiles {
		p := filepath.Join(dir, name)
		info, err := os.Stat(p)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		backup.files = append(backup.files, fileBackup{path: p, data: data, mode: info.Mode().Perm()})
	}
	return backup, nil
}

// Restore writes the backed up files back to the project directory.
// Nothing is done if there was no previous installation.
func (b *ProjectBackup) Restore() error {
	var errs []error
	for _, f := range b.files {
		slog.Info("Restoring previous project file.", "path", f.path)
		if err := os.WriteFile(f.path, f.data, f.mode); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
