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

func Test_PodmanMultipleExitCodes(t *testing.T) {
	output := `
['podman', '--version', '']
using podman version: 4.3.1
** excluding:  set()
podman volume inspect nodered_node_red_data || podman volume create nodered_node_red_data
['podman', 'volume', 'inspect', 'nodered_node_red_data']
['podman', 'network', 'exists', 'tedge']
podman run --name=nodered_nodered_1 -d --label io.podman.compose.config-hash=123 --label io.podman.compose.project=nodered --label io.podman.compose.version=0.0.1 --label com.docker.compose.project=nodered --label com.docker.compose.project.working_dir=/var/tedge-container-plugin/compose/nodered --label com.docker.compose.project.config_files=docker-compose.yaml --label com.docker.compose.container-number=1 --label com.docker.compose.service=nodered -e NODE_RED_ENABLE_PROJECTS=false -e TEDGE_MQTT_HOST=host.containers.internal -e TEDGE_MQTT_PORT=1884 -v nodered_node_red_data:/data --net tedge --network-alias nodered -p 1880:1880 docker.io/nodered/node-red:4.0.3-22-minimal
Error: creating container storage: the container name "nodered_nodered_1" is already in use by 557756c5b134746b00c128f36f01627635d2f01b491127ea4a206e8381af0310. You have to remove that container to be able to reuse that name: that name is already in use
exit code: 125
podman start nodered_nodered_1
exit code: 0
`
	if CheckPodmanComposeError(output) != nil {
		t.Errorf("did not expect an error")
	}
}
