package container

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func writeComposeFile(t *testing.T, dir string, name string, contents string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReadComposeSettings(t *testing.T) {
	cases := []struct {
		name          string
		contents      string
		removeVolumes *bool
		removeTimeout *time.Duration
		wantErr       bool
	}{
		{
			name:     "no settings",
			contents: "services:\n  app:\n    image: hello-world\n",
		},
		{
			name:     "empty settings",
			contents: "x-tedge:\nservices:\n  app:\n    image: hello-world\n",
		},
		{
			name:          "keep volumes",
			contents:      "x-tedge:\n  remove_volumes: false\n",
			removeVolumes: new(false),
		},
		{
			name:          "remove volumes",
			contents:      "x-tedge:\n  remove_volumes: true\n",
			removeVolumes: new(true),
		},
		{
			name:          "timeout as duration",
			contents:      "x-tedge:\n  remove_timeout: 1m30s\n",
			removeTimeout: new(90 * time.Second),
		},
		{
			name:          "timeout as seconds",
			contents:      "x-tedge:\n  remove_timeout: 45\n",
			removeTimeout: new(45 * time.Second),
		},
		{
			name:          "all settings",
			contents:      "x-tedge:\n  remove_volumes: false\n  remove_timeout: 0\n",
			removeVolumes: new(false),
			removeTimeout: new(time.Duration(0)),
		},
		{
			name:     "unknown settings are ignored",
			contents: "x-tedge:\n  future_setting: 1\n",
		},
		{
			name:     "invalid volumes value",
			contents: "x-tedge:\n  remove_volumes: maybe\n",
			wantErr:  true,
		},
		{
			name:     "invalid timeout value",
			contents: "x-tedge:\n  remove_timeout: soon\n",
			wantErr:  true,
		},
		{
			name:     "negative timeout",
			contents: "x-tedge:\n  remove_timeout: -5s\n",
			wantErr:  true,
		},
		{
			name:     "settings not a mapping",
			contents: "x-tedge: true\n",
			wantErr:  true,
		},
		{
			name:     "invalid yaml",
			contents: "x-tedge: [\n",
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := writeComposeFile(t, t.TempDir(), "docker-compose.yaml", tc.contents)
			settings, err := ReadComposeSettings(p)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.removeVolumes, settings.RemoveVolumes)
			if tc.removeTimeout == nil {
				assert.Nil(t, settings.RemoveTimeout)
			} else if assert.NotNil(t, settings.RemoveTimeout) {
				assert.Equal(t, *tc.removeTimeout, time.Duration(*settings.RemoveTimeout))
			}
		})
	}
}

func TestFindComposeFiles(t *testing.T) {
	dir := t.TempDir()
	assert.Empty(t, FindComposeFiles(dir))

	// an override file is not used without a main compose file
	writeComposeFile(t, dir, "docker-compose.override.yml", "")
	assert.Empty(t, FindComposeFiles(dir))

	writeComposeFile(t, dir, "docker-compose.yaml", "")
	assert.Equal(t, []string{
		filepath.Join(dir, "docker-compose.yaml"),
		filepath.Join(dir, "docker-compose.override.yml"),
	}, FindComposeFiles(dir))

	writeComposeFile(t, dir, "docker-compose.yml", "")
	writeComposeFile(t, dir, "compose.override.yaml", "")
	assert.Equal(t, []string{
		filepath.Join(dir, "docker-compose.yml"),
		filepath.Join(dir, "compose.override.yaml"),
	}, FindComposeFiles(dir))

	writeComposeFile(t, dir, "compose.yaml", "")
	assert.Equal(t, filepath.Join(dir, "compose.yaml"), FindComposeFiles(dir)[0])
}

func TestReadComposeSettingsWithOverride(t *testing.T) {
	dir := t.TempDir()
	main := writeComposeFile(t, dir, "compose.yaml", "x-tedge:\n  remove_volumes: true\n  remove_timeout: 20s\n")
	override := writeComposeFile(t, dir, "compose.override.yaml", "x-tedge:\n  remove_volumes: false\n")

	settings, err := ReadComposeSettings(main, override)
	assert.NoError(t, err)
	assert.Equal(t, new(false), settings.RemoveVolumes)
	assert.Equal(t, 20*time.Second, time.Duration(*settings.RemoveTimeout))

	// an invalid override file is an error
	writeComposeFile(t, dir, "compose.override.yaml", "x-tedge:\n  remove_volumes: nope\n")
	_, err = ReadComposeSettings(main, override)
	assert.ErrorContains(t, err, "compose.override.yaml")
}

func TestReadComposeSettingsUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	p := writeComposeFile(t, dir, "compose.yaml", "x-tedge:\n  remove_volume: false\n  remove_timeout: 5\n  keep_volumes: true\n")
	settings, err := ReadComposeSettings(p)
	assert.NoError(t, err)
	assert.Nil(t, settings.RemoveVolumes)
	assert.Equal(t, []string{"remove_volume", "keep_volumes"}, settings.UnknownKeys)
	assert.ElementsMatch(t, []string{"remove_volumes", "remove_timeout"}, composeSettingsKeys())
}

func TestReadComposeSettingsAlias(t *testing.T) {
	p := writeComposeFile(t, t.TempDir(), "compose.yaml", "x-defaults: &defaults\n  remove_volumes: false\nx-tedge: *defaults\n")
	settings, err := ReadComposeSettings(p)
	assert.NoError(t, err)
	assert.Equal(t, new(false), settings.RemoveVolumes)
	assert.Empty(t, settings.UnknownKeys)
}

func TestParseDuration(t *testing.T) {
	cases := map[string]time.Duration{
		"":      0,
		"0":     0,
		"30":    30 * time.Second,
		" 30 ":  30 * time.Second,
		"1.5":   1500 * time.Millisecond,
		"30s":   30 * time.Second,
		"1m30s": 90 * time.Second,
	}
	for in, want := range cases {
		got, err := ParseDuration(in)
		assert.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	for _, in := range []string{"abc", "-5", "-5s"} {
		_, err := ParseDuration(in)
		assert.Error(t, err, in)
	}
}

func TestResolveComposeDownOptions(t *testing.T) {
	defaults := ComposeDownOptions{RemoveVolumes: true, RemoveTimeout: 20 * time.Second}

	t.Run("no compose file uses defaults", func(t *testing.T) {
		assert.Equal(t, defaults, resolveComposeDownOptions("app", t.TempDir(), defaults))
	})

	t.Run("compose file without settings uses defaults", func(t *testing.T) {
		dir := t.TempDir()
		writeComposeFile(t, dir, "docker-compose.yaml", "services: {}\n")
		assert.Equal(t, defaults, resolveComposeDownOptions("app", dir, defaults))
	})

	t.Run("compose file settings take precedence", func(t *testing.T) {
		dir := t.TempDir()
		writeComposeFile(t, dir, "docker-compose.yaml", "x-tedge:\n  remove_volumes: false\n  remove_timeout: 2m\n")
		assert.Equal(t, ComposeDownOptions{RemoveVolumes: false, RemoveTimeout: 2 * time.Minute}, resolveComposeDownOptions("app", dir, defaults))
	})

	t.Run("compose file can enable volume removal", func(t *testing.T) {
		dir := t.TempDir()
		writeComposeFile(t, dir, "docker-compose.yaml", "x-tedge:\n  remove_volumes: true\n")
		assert.Equal(t, ComposeDownOptions{RemoveVolumes: true}, resolveComposeDownOptions("app", dir, ComposeDownOptions{}))
	})

	t.Run("invalid settings keep volumes", func(t *testing.T) {
		dir := t.TempDir()
		writeComposeFile(t, dir, "docker-compose.yaml", "x-tedge:\n  remove_volumes: nope\n")
		assert.Equal(t, ComposeDownOptions{RemoveVolumes: false, RemoveTimeout: 20 * time.Second}, resolveComposeDownOptions("app", dir, defaults))
	})
}

func TestComposeDownOptionsArgs(t *testing.T) {
	cases := []struct {
		name     string
		opts     ComposeDownOptions
		stopArgs []string
		downArgs []string
	}{
		{
			name:     "defaults",
			opts:     ComposeDownOptions{},
			stopArgs: []string{"stop"},
			downArgs: []string{"down", "--remove-orphans"},
		},
		{
			name:     "remove volumes",
			opts:     ComposeDownOptions{RemoveVolumes: true},
			stopArgs: []string{"stop"},
			downArgs: []string{"down", "--remove-orphans", "--volumes"},
		},
		{
			name:     "timeout",
			opts:     ComposeDownOptions{RemoveVolumes: true, RemoveTimeout: 30 * time.Second},
			stopArgs: []string{"stop", "--timeout", "30"},
			downArgs: []string{"down", "--remove-orphans", "--volumes", "--timeout", "30"},
		},
		{
			name:     "timeout is rounded up to whole seconds",
			opts:     ComposeDownOptions{RemoveTimeout: 1500 * time.Millisecond},
			stopArgs: []string{"stop", "--timeout", "2"},
			downArgs: []string{"down", "--remove-orphans", "--timeout", "2"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.stopArgs, tc.opts.StopArgs())
			assert.Equal(t, tc.downArgs, tc.opts.DownArgs())
		})
	}
}
