package container

// LibPodInspectResponse is a minimal subset of the JSON returned by
// GET /libpod/containers/{name}/json.  Only the fields needed to recover
// namespace settings that the Docker-compat API normalises away are captured
// here.  Add fields as needed; unused fields are silently ignored during
// JSON deserialisation.
type LibPodInspectResponse struct {
	HostConfig LibPodHostConfig `json:"HostConfig"`
}

// LibPodHostConfig mirrors the HostConfig portion of a libpod container inspect
// response.  Unlike the Docker-compat API, podman returns the original value for
// fields like UsernsMode (e.g. "keep-id") rather than a normalised substitute
// (e.g. "private").
type LibPodHostConfig struct {
	// UsernsMode is the user-namespace mode as stored by podman, e.g. "" (host),
	// "private", "keep-id", "keep-id:uid=X,gid=Y", "nomap", etc.
	// NOTE: podman 4.x normalises "keep-id" to "private" here; use IDMappings
	// to detect the original mode when this field cannot be trusted.
	UsernsMode string            `json:"UsernsMode"`
	IDMappings *LibPodIDMappings `json:"IDMappings,omitempty"`
}

// LibPodIDMappings holds the UID/GID mapping tables for a container's user
// namespace. Each entry is in "containerID:hostID:size" format.
type LibPodIDMappings struct {
	UIDMap []string `json:"UIDMap"`
	GIDMap []string `json:"GIDMap"`
}
