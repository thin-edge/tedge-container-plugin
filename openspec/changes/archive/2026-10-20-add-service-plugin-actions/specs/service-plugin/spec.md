## ADDED Requirements

### Requirement: Service plugin entry points
The package SHALL install executable service plugins at `/usr/share/tedge/service-plugins/container` and `/usr/share/tedge/service-plugins/container-group`, owned by `root:root` with mode `0755`. Each SHALL delegate to `tedge-container service-plugins <type>`, passing all arguments through unchanged.

#### Scenario: Plugins installed by the package
- **WHEN** the tedge-container-plugin package is installed
- **THEN** both plugin files exist, are owned by `root:root`, have mode `0755`, and are not writable by the `tedge` user

#### Scenario: Invoked by tedge service
- **WHEN** `sudo tedge service restart app1 --type container` is run
- **THEN** `tedge-container service-plugins container restart app1` is executed

### Requirement: Invocation contract
The service plugin SHALL accept exactly two positional arguments, `<action> <service-name>`, and SHALL NOT interpret either one as a flag.

#### Scenario: Wrong number of arguments
- **WHEN** the plugin is invoked with fewer or more than two arguments
- **THEN** it exits with a non-zero code other than `2` and prints the usage to stderr

### Requirement: Supported actions
The service plugin SHALL support the actions `start`, `stop`, `restart`, `pause` and `unpause` for both the `container` and the `container-group` types, using the container engine API (docker or podman).

#### Scenario: Restart a container
- **WHEN** `service-plugins container restart app1` is run and the container `app1` is running
- **THEN** the container is restarted, so its start time changes, and the plugin exits `0`

#### Scenario: Stop a container
- **WHEN** `service-plugins container stop app1` is run and `app1` is running
- **THEN** `app1` is stopped and the plugin exits `0`

#### Scenario: Start a stopped container
- **WHEN** `service-plugins container start app1` is run and `app1` is stopped
- **THEN** `app1` is running and the plugin exits `0`

#### Scenario: Pause and unpause a container
- **WHEN** `service-plugins container pause app1` and then `service-plugins container unpause app1` are run
- **THEN** `app1` is first paused and then running again, and both invocations exit `0`

#### Scenario: Action already in the requested state
- **WHEN** `start` is requested for a running container, or `stop` for a stopped one
- **THEN** the plugin exits `0`

### Requirement: Unsupported actions
The service plugin SHALL exit with code `2` for any action it does not implement, and print to stderr a message naming the action and the service type.

#### Scenario: Unknown action
- **WHEN** `service-plugins container enable app1` is run
- **THEN** the plugin exits `2`, stderr names the unsupported action `enable`, and the container is not changed

### Requirement: Failure reporting
The service plugin SHALL exit with a non-zero code other than `2` when a supported action cannot be completed, and print the reason to stderr.

#### Scenario: Unknown container
- **WHEN** `service-plugins container restart does-not-exist` is run
- **THEN** the plugin exits `1` and stderr says that the container was not found

#### Scenario: Engine error
- **WHEN** the container engine returns an error for the requested operation
- **THEN** the plugin exits `1` and the engine's error message is on stderr

### Requirement: Container name resolution
For the `container` type, the service name SHALL be resolved as a container name or ID.

#### Scenario: Resolve by container name
- **WHEN** the service name is `app1` and a container named `app1` exists
- **THEN** the action is applied to that container

### Requirement: Container-group name resolution
For the `container-group` type, the service name SHALL have the format `<project>@<service>`. The project part SHALL be resolved to the compose project label in the same way as the container-group log plugin, so that it works with either `container_group.use_module_name` setting. The action SHALL be applied to the one container of that compose service.

#### Scenario: Restart a compose service
- **WHEN** `service-plugins container-group restart nginx@web` is run and project `nginx` has a service `web`
- **THEN** only the `web` container of project `nginx` is restarted, and the plugin exits `0`

#### Scenario: Module name differs from compose project name
- **WHEN** the service name uses the module name stored at install time and the compose project has a different name
- **THEN** the project is resolved and the action is applied to the matching container

#### Scenario: Invalid name format
- **WHEN** `service-plugins container-group restart nginx` is run, without an `@`
- **THEN** the plugin exits `1` and stderr says that the expected format is `<project>@<service>`

#### Scenario: Ambiguous compose service
- **WHEN** the compose service matches more than one container, e.g. because it is scaled
- **THEN** the plugin exits `1`, lists the matching containers on stderr, and changes none of them

### Requirement: Refuse actions on the hosting container
The service plugin SHALL refuse every action on the container in which the plugin itself runs. It SHALL exit `1` with a reason saying that the container hosting thin-edge.io cannot be acted on.

#### Scenario: Restart of own container
- **WHEN** the plugin runs inside container `tedge` and `service-plugins container restart tedge` is run
- **THEN** the plugin exits `1`, prints the refusal reason on stderr, and the container is not restarted

#### Scenario: Running on the host
- **WHEN** the plugin does not run inside a container
- **THEN** no container is refused on this ground

### Requirement: Output free of workflow markers
The service plugin SHALL NOT print the `:::begin-tedge:::` or `:::end-tedge:::` markers on stdout or stderr.

#### Scenario: Container output containing markers
- **WHEN** any action is run
- **THEN** the plugin's output contains no workflow state markers
