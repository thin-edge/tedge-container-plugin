# service-action-declaration Specification

## Purpose
How the tedge-container-plugin service declares the `cmd/<action>` capabilities of the container services it registers, so that the actions are offered as Cumulocity service commands.
## Requirements
### Requirement: Declare actions of registered services
When the service actions feature is active, the daemon SHALL publish a retained `{}` with QoS 1 on `te/<service-topic-id>/cmd/<action>` for each configured action of every `container` and `container-group` service it registers.

#### Scenario: Registration failed
- **WHEN** the registration of a container service fails
- **THEN** no actions are declared for it in that cycle

#### Scenario: New container registered
- **WHEN** a container `app1` is registered as a service while the feature is active with the default actions
- **THEN** retained `{}` messages exist on `te/device/main/service/app1/cmd/start`, `.../cmd/stop` and `.../cmd/restart`
- **AND** the Cumulocity service `app1` gets `c8y_SupportedServiceCommands` = `["RESTART", "START", "STOP"]`

#### Scenario: Container-group service registered
- **WHEN** the compose service `nginx@web` is registered
- **THEN** the configured actions are declared on that service's topic

#### Scenario: Restart a container from Cumulocity
- **WHEN** a `c8y_ServiceCommand` operation with command `RESTART` is created for the service `app1`
- **THEN** the container `app1` is restarted and the operation becomes `SUCCESSFUL`

#### Scenario: Unsupported action from Cumulocity
- **WHEN** a `c8y_ServiceCommand` operation is created for an action that is not declared
- **THEN** the operation becomes `FAILED` and the container is not changed

### Requirement: Declare once per service and process
The daemon SHALL declare the actions of a service only once per process lifetime. It SHALL NOT republish them on later update or reconcile cycles while the service stays registered. A service that is deregistered and registered again SHALL be declared again.

#### Scenario: Repeated update cycles
- **WHEN** several update cycles run while `app1` stays registered
- **THEN** the `cmd/<action>` topics of `app1` are published only during the first cycle

#### Scenario: Container re-created
- **WHEN** `app1` is removed, its service deregistered, and a new container `app1` created
- **THEN** the actions are declared again for `app1`

#### Scenario: Failed publish
- **WHEN** publishing a declaration fails
- **THEN** the service is declared again on the next update cycle

### Requirement: Clean up on deregistration
Service deregistration SHALL leave no `cmd/<action>` capability topics of that service behind.

#### Scenario: Container removed
- **WHEN** the container `app1` is removed and its service deregistered
- **THEN** no retained message is left on any `te/device/main/service/app1/cmd/<action>` topic

### Requirement: Configurable actions
The `service_commands.actions` setting SHALL select which actions are declared. Its default SHALL be `["start", "stop", "restart"]`. Only the actions for which tedge-agent ships a workflow (`start`, `stop`, `restart`) SHALL be declarable. Other entries SHALL be ignored, with a warning in the log. Declarable actions that are not configured SHALL be cleared when a service is first declared. Actions that are not declarable SHALL never be published or cleared by the daemon.

#### Scenario: Action without a shipped workflow
- **WHEN** `service_commands.actions = ["start", "stop", "restart", "pause"]`
- **THEN** only `start`, `stop` and `restart` are declared, and a warning names `pause` as not declarable

#### Scenario: Manually declared action is kept
- **WHEN** a user has declared `cmd/pause` on a service along with a custom workflow
- **THEN** the daemon neither clears nor republishes `cmd/pause`

#### Scenario: Narrowed action list after restart
- **WHEN** `restart` was previously declared and the daemon restarts with `service_commands.actions = ["start", "stop"]`
- **THEN** `cmd/restart` of each service is cleared and `RESTART` is removed from `c8y_SupportedServiceCommands`

#### Scenario: Unknown action configured
- **WHEN** `service_commands.actions` contains `reboot`
- **THEN** `reboot` is not declared and a warning is logged

### Requirement: Feature activation
The `service_commands.enabled` setting SHALL accept `auto` (the default), `true` and `false`.
- With `auto`, the feature SHALL be active only when `tedge config get service.plugin_paths` succeeds and one of the returned directories contains the `container` service plugin.
- With `true`, the feature SHALL be active without the check.
- With `false`, it SHALL be inactive.

The daemon SHALL log at info level whether the feature is active, and why.

#### Scenario: thin-edge.io without service plugin support
- **WHEN** `enabled = "auto"` and `tedge config get service.plugin_paths` fails
- **THEN** no `cmd/<action>` topics are published for any container service, and the log says that service actions are inactive because thin-edge.io lacks service plugin support

#### Scenario: Plugin not installed in configured paths
- **WHEN** `enabled = "auto"` and none of the configured `service.plugin_paths` directories contains the `container` plugin
- **THEN** no actions are declared, and the reason is logged

#### Scenario: Supported thin-edge.io
- **WHEN** `enabled = "auto"`, the setting exists, and `/usr/share/tedge/service-plugins/container` is present in a configured path
- **THEN** actions are declared

#### Scenario: Forced on
- **WHEN** `enabled = true` and the `tedge` CLI is not available to the daemon
- **THEN** actions are declared

#### Scenario: Disabled
- **WHEN** `enabled = false`
- **THEN** no actions are declared, and the declarable action topics of each registered service are cleared once per process

### Requirement: No actions on the hosting container
The daemon SHALL NOT declare actions on the service of the container it runs in, and SHALL clear any known action topics of that service.

#### Scenario: Running in tedge-container-bundle
- **WHEN** the daemon runs inside container `tedge`, which is also registered as a service
- **THEN** no `cmd/<action>` topics are retained for the service `tedge`, and Cumulocity offers no service commands for it

#### Scenario: Running on the host
- **WHEN** the daemon does not run inside a container
- **THEN** actions are declared for every registered container service

