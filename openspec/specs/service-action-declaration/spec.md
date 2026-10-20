# service-action-declaration Specification

## Purpose
How the tedge-container-plugin service declares the actions of the container services it registers in their `cmd/control` capability (the thin-edge.io [service control](https://thin-edge.github.io/thin-edge.io/next/references/agent/service-control/) contract), so that the actions are offered as Cumulocity service commands.
## Requirements
### Requirement: Declare actions of registered services
When the service actions feature is active, the daemon SHALL publish a retained `{"actions": [...]}` message with QoS 1 on `te/<service-topic-id>/cmd/control`, listing the configured actions in their configured order, for every `container` and `container-group` service it registers. When no actions are to be declared on a service, the daemon SHALL publish a retained empty message on that topic instead, which clears the actions declared before.

#### Scenario: Registration failed
- **WHEN** the registration of a container service fails
- **THEN** no actions are declared for it in that cycle

#### Scenario: New container registered
- **WHEN** a container `app1` is registered as a service while the feature is active with the default actions
- **THEN** a retained `{"actions":["start","stop","restart"]}` message exists on `te/device/main/service/app1/cmd/control`
- **AND** the Cumulocity service `app1` gets `c8y_SupportedServiceCommands` = `["start", "stop", "restart"]`

#### Scenario: Container-group service registered
- **WHEN** the compose service `nginx@web` is registered
- **THEN** the configured actions are declared on that service's topic

#### Scenario: Restart a container from Cumulocity
- **WHEN** a `c8y_ServiceCommand` operation with command `restart` is created for the service `app1`
- **THEN** the container `app1` is restarted and the operation becomes `SUCCESSFUL`

#### Scenario: Unsupported action from Cumulocity
- **WHEN** a `c8y_ServiceCommand` operation is created for an action that is not declared
- **THEN** the operation becomes `FAILED` and the container is not changed

### Requirement: Declare once per service and process
The daemon SHALL declare the actions of a service only once per process lifetime. It SHALL NOT republish them on later update or reconcile cycles while the service stays registered. A service that is deregistered and registered again SHALL be declared again.

#### Scenario: Repeated update cycles
- **WHEN** several update cycles run while `app1` stays registered
- **THEN** the `cmd/control` topic of `app1` is published only during the first cycle

#### Scenario: Container re-created
- **WHEN** `app1` is removed, its service deregistered, and a new container `app1` created
- **THEN** the actions are declared again for `app1`

#### Scenario: Failed publish
- **WHEN** publishing a declaration fails
- **THEN** the service is declared again on the next update cycle

### Requirement: Clean up on deregistration
Service deregistration SHALL leave no `cmd/control` capability of that service behind.

#### Scenario: Container removed
- **WHEN** the container `app1` is removed and its service deregistered
- **THEN** no retained message is left on `te/device/main/service/app1/cmd/control`

### Requirement: Configurable actions
The `service_commands.actions` setting SHALL select which actions are declared, and in which order. Its default SHALL be `["start", "stop", "restart"]`. Every action supported by the service plugins (`start`, `stop`, `restart`, `pause`, `unpause`) SHALL be declarable, as the `control` workflow shipped by tedge-agent runs any action through the service plugin. Other entries SHALL be ignored, with a warning in the log. Duplicates SHALL be declared once.

#### Scenario: Custom actions
- **WHEN** `service_commands.actions = ["restart", "pause", "unpause"]`
- **THEN** `{"actions":["restart","pause","unpause"]}` is declared, and a `c8y_ServiceCommand` operation with command `pause` pauses the container

#### Scenario: Narrowed action list after restart
- **WHEN** `restart` was previously declared and the daemon restarts with `service_commands.actions = ["start", "stop"]`
- **THEN** `{"actions":["start","stop"]}` is declared on each service and `restart` is removed from `c8y_SupportedServiceCommands`

#### Scenario: Unknown action configured
- **WHEN** `service_commands.actions` contains `reboot`
- **THEN** `reboot` is not declared and a warning is logged

### Requirement: Feature activation
The `service_commands.enabled` setting SHALL accept `auto` (the default), `true` and `false`.
- With `auto`, the feature SHALL be active only when `tedge config get c8y.enable.service_control` succeeds (thin-edge.io supports the `control` command), `tedge config get service.plugin_paths` succeeds, and one of the returned directories contains the `container` service plugin.
- With `true`, the feature SHALL be active without the check.
- With `false`, it SHALL be inactive.

The daemon SHALL log at info level whether the feature is active, and why.

#### Scenario: thin-edge.io without service plugin support
- **WHEN** `enabled = "auto"` and `tedge config get service.plugin_paths` fails
- **THEN** no `cmd/control` topic is published for any container service, and the log says that service actions are inactive because thin-edge.io lacks service plugin support

#### Scenario: thin-edge.io without the control command
- **WHEN** `enabled = "auto"` and `tedge config get c8y.enable.service_control` fails (e.g. a thin-edge.io version which declared one `cmd/<action>` topic per action)
- **THEN** no `cmd/control` topic is published for any container service, and the log says that thin-edge.io lacks service control support

#### Scenario: Plugin not installed in configured paths
- **WHEN** `enabled = "auto"` and none of the configured `service.plugin_paths` directories contains the `container` plugin
- **THEN** no actions are declared, and the reason is logged

#### Scenario: Supported thin-edge.io
- **WHEN** `enabled = "auto"`, both settings exist, and `/usr/share/tedge/service-plugins/container` is present in a configured path
- **THEN** actions are declared

#### Scenario: Forced on
- **WHEN** `enabled = true` and the `tedge` CLI is not available to the daemon
- **THEN** actions are declared

#### Scenario: Disabled
- **WHEN** `enabled = false`
- **THEN** no actions are declared, and the `cmd/control` topic of each registered service is cleared once per process

### Requirement: No actions on the hosting container
The daemon SHALL NOT declare actions on the service of the container it runs in, and SHALL clear the `cmd/control` topic of that service.

#### Scenario: Running in tedge-container-bundle
- **WHEN** the daemon runs inside container `tedge`, which is also registered as a service
- **THEN** no `cmd/control` capability is retained for the service `tedge`, and Cumulocity offers no service commands for it

#### Scenario: Running on the host
- **WHEN** the daemon does not run inside a container
- **THEN** actions are declared for every registered container service

