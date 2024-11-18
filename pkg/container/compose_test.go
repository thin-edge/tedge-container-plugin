package container

import "testing"

func Test_Podman(t *testing.T) {
	output := `
podman-compose build
['podman', '--version', '']
using podman version: 4.3.1
podman build -t app1_app1 -f ./Dockerfile .
STEP 1/2: FROM nginx
Error: creating build container: short-name "nginx" did not resolve to an alias and no unqualified-search registries are defined in "/etc/containers/registries.conf"
exit code: 125
`
	if CheckPodmanComposeError(output) == nil {
		t.Fail()
	}
}
