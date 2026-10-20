## Why

thin-edge.io now ships generic **service commands**: a service declares actions on `te/<service>/cmd/<action>`, the Cumulocity mapper exposes them as `c8y_SupportedServiceCommands`, and `tedge-agent`'s shipped workflow runs `sudo tedge service <action> <name> --type <type>`. For any service type other than `service`, `tedge service` delegates to a [service plugin](https://thin-edge.github.io/thin-edge.io/next/references/service-plugin-api/) named after the type, at `<service.plugin_paths>/<type>`.

The services this plugin registers have the types `container` and `container-group`, but no service plugin exists for them. Operators therefore cannot start, stop or restart a container from the Cumulocity **Services > Commands** tab, even though the plugin already knows how to do all of it (see `tools container-restart`).

## What Changes

- **Service plugin commands**: add `tedge-container service-plugins container <action> <name>` and `tedge-container service-plugins container-group <action> <name>`. They implement the Service Plugin API contract:
  - actions `start`, `stop`, `restart`, `pause` and `unpause`, run against the container engine API (docker or podman)
  - exit `0` on success, `2` for an action the plugin does not support, any other non-zero code on failure, with the reason on stderr
  - `container`: the service name is the container name
  - `container-group`: the service name is `<project>@<service>`. It is resolved the same way as the log plugin does it (`ParseContainerGroup` → `ResolveComposeProjectName` → `LookupProject`), so both `use_module_name` settings work
  - the plugin never prints the `:::begin-tedge:::` / `:::end-tedge:::` markers
- **Packaging**: ship the shims `/usr/share/tedge/service-plugins/container` and `/usr/share/tedge/service-plugins/container-group`, root-owned with mode `0755`. They follow the existing log-plugin shims (`exec tedge-container service-plugins <type> "$@"`). The plugin directory must not be writable by the `tedge` user.
- **Action declaration**: when the plugin registers a container or container-group service, it also publishes a retained `{}` on `te/<service-topic-id>/cmd/<action>` for each enabled action. Services are deregistered through the entity API, which already clears these topics.
- **Feature gating**: actions are declared only when the installed thin-edge.io supports service plugins. The check is that `tedge config get service.plugin_paths` succeeds and one of those paths holds the plugin's file. On an older thin-edge.io (≤ 2.0.x), no actions are declared, so Cumulocity never offers a command that would fail.
- **Configuration**: a new `[service_commands]` section in `tedge-container-plugin.toml`:
  - `enabled` (default `"auto"`: declare only when thin-edge.io supports service plugins; `true` and `false` force it on or off)
  - `actions` (default `["start", "stop", "restart"]`), to narrow down what is declared. Only actions with a tedge-agent workflow can be declared. `pause` and `unpause` are supported by the plugin, but need a custom workflow and a manual declaration, as documented in the README
- **Self-protection**: no actions are declared on the container that hosts the plugin itself, e.g. `tedge-container-bundle`. Stopping it would cut the device off. Restarting it would kill `tedge-agent` in the middle of the workflow, and `tedge-agent` would re-run the `execute` step on resume. The service plugin also refuses these actions with exit code `1` and a clear reason, as a second line of defense.

## Capabilities

### New Capabilities
- `service-plugin`: the `container` and `container-group` service plugins. Covers the CLI contract (arguments, supported actions, exit codes, output), name resolution, the behavior on the engine for each action, the refusal on the plugin's own container, and the packaging and permissions of the plugin files.
- `service-action-declaration`: how and when the plugin declares the `cmd/<action>` capabilities of the services it registers. Covers feature detection of thin-edge.io support, the `[service_commands]` configuration, exclusion of the plugin's own container, and avoiding republishing on every update cycle.

### Modified Capabilities
<!-- None: openspec/specs/ has no existing specs. -->

## Impact

- **Code**:
  - new `cli/service_plugins/` package (one command for each service type), modelled on `cli/log_plugins/`
  - `pkg/app/app.go` `doUpdate`: declare actions after registration
  - `pkg/cli` / config: the new `[service_commands]` settings
  - small helper in `pkg/tedge` to publish the capability topics
- **Packaging**:
  - `.goreleaser.yaml`: two new files under `/usr/share/tedge/service-plugins/`
  - `packaging/service-plugins/`: the shims
  - `packaging/config.toml`: the new section
- **Runtime**:
  - `tedge service` runs the plugin as root via `sudo`. The plugin then reads `/etc/tedge/plugins/tedge-container-plugin.toml` to find the engine socket. This works the same way as the log plugins, and needs checking for rootless podman, where the environment differs under `sudo`.
  - Up to 3 extra retained MQTT messages for each container, published once per service and not on every reconcile.
- **Dependencies**: requires a thin-edge.io release with the Service Plugin API, i.e. `tedge service` and `service.plugin_paths`. Older releases keep working, just without service actions.
- **Tests**: new Robot Framework suite `tests/service-commands.robot`:
  - start, stop and restart of a `container` and a `container-group` service from Cumulocity (`c8y_ServiceCommand`)
  - the `c8y_SupportedServiceCommands` content
  - an unsupported action exits `2`
  - an unknown service fails
  - no actions are declared on the plugin's own container
- **Docs**: README section on service commands and the new config settings.
