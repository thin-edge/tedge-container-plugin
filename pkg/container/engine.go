package container

import "strings"

// EngineType identifies the container engine backend.
// Use the typed constants rather than raw string comparisons throughout the codebase.
type EngineType string

const (
	EngineDocker  EngineType = "docker"
	EnginePodman  EngineType = "podman"
	EngineUnknown EngineType = "unknown"
)

// EngineCapabilities describes the features available from the detected engine.
// Adding a new engine in the future means adding fields here (and a new case in
// detectEngineCapabilities) without touching any call sites.
type EngineCapabilities struct {
	// Type is the engine variant, e.g. EnginePodman or EngineDocker.
	Type EngineType

	// HasLibPodAPI indicates that the libpod REST API is available at the same
	// socket. Only true for podman instances.
	HasLibPodAPI bool

	// Version is the server version string reported by the engine (e.g. "4.6.1").
	Version string
}

// detectEngineCapabilities maps the engine name string returned by the docker/compat
// Info() call to an EngineCapabilities value.
// engineName is compared case-insensitively.
func detectEngineCapabilities(engineName string) EngineCapabilities {
	lower := strings.ToLower(engineName)
	switch {
	case strings.Contains(lower, "podman"):
		return EngineCapabilities{
			Type:         EnginePodman,
			HasLibPodAPI: true,
		}
	case strings.Contains(lower, "docker"):
		return EngineCapabilities{
			Type:         EngineDocker,
			HasLibPodAPI: false,
		}
	default:
		return EngineCapabilities{
			Type:         EngineUnknown,
			HasLibPodAPI: false,
		}
	}
}
