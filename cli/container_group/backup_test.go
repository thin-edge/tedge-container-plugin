package container_group

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestProjectBackupRestore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "docker-compose.yaml", "image: app:1.0.3")
	writeFile(t, dir, "version", "1.0.3")
	writeFile(t, dir, "other.txt", "untouched")

	backup, err := BackupProject(dir)
	if err != nil {
		t.Fatal(err)
	}

	// simulate a failed install which already replaced the compose file
	writeFile(t, dir, "docker-compose.yaml", "image: app:1.0.4")
	writeFile(t, dir, "other.txt", "changed")

	if err := backup.Restore(); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dir, "docker-compose.yaml"); got != "image: app:1.0.3" {
		t.Errorf("compose file not restored. got=%q", got)
	}
	if got := readFile(t, dir, "version"); got != "1.0.3" {
		t.Errorf("version file changed. got=%q", got)
	}
	if got := readFile(t, dir, "other.txt"); got != "changed" {
		t.Errorf("files which are not part of the backup should not be touched. got=%q", got)
	}
}

func TestProjectBackupFirstInstall(t *testing.T) {
	dir := t.TempDir()

	backup, err := BackupProject(dir)
	if err != nil {
		t.Fatal(err)
	}

	// nothing to restore, so the files of a failed first install are left as is
	writeFile(t, dir, "docker-compose.yaml", "image: app:1.0.4")
	if err := backup.Restore(); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dir, "docker-compose.yaml"); got != "image: app:1.0.4" {
		t.Errorf("unexpected change. got=%q", got)
	}
}
