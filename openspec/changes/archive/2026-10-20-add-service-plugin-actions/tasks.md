## 1. Container client

- [x] 1.1 Add `PauseContainer` and `UnpauseContainer` to `ContainerClient` (`pkg/container/container.go`)
- [x] 1.2 Extract the container-group resolution into a shared helper `LookupGroupService(ctx, name)` that rejects an invalid format and matches the compose service label exactly, including stopped containers. The check for exactly one match is done in the service plugin, so the log plugin keeps using the first match
- [x] 1.3 Refactor the container-group log plugin to use the shared helper. It now also finds the logs of stopped containers and no longer matches a container name by substring
- [x] 1.4 Add a unit test for the rejection of invalid names. The lookup itself needs an engine and is covered by the system tests

## 2. Service plugin commands

- [x] 2.1 Create the `cli/service_plugins` package with `service-plugins container` and `service-plugins container-group` commands. Each takes `ExactArgs(2)` and sets `DisableFlagParsing`
- [x] 2.2 Implement the action table (`start`, `stop`, `restart`, `pause`, `unpause`) and the sentinel `ErrUnsupportedAction`
- [x] 2.3 Resolve `container` names through inspect by name or ID, and `container-group` names through the helper from 1.2
- [x] 2.4 Refuse actions on the hosting container: call `Self()`, compare container IDs, and exit 1 with the reason
- [x] 2.5 Register `service_plugins.NewCommand` in `cmd/root.go`
- [x] 2.6 Map `ErrUnsupportedAction` to exit code `2` by returning `cli.ExitCodeError`, which `cmd.Execute` already maps to the exit code. Add `Unwrap` to `ExitCodeError` so `errors.Is` works
- [x] 2.7 Add unit tests for the dispatch and exit-code mapping: unknown action → 2, wrong arg count → 1

## 3. Configuration

- [x] 3.1 Add the defaults `service_commands.enabled = "auto"` and `service_commands.actions = ["start","stop","restart"]`, plus the accessors `ServiceCommandsMode()` and `ServiceCommandsActions()` in `pkg/cli/cli.go`
- [x] 3.2 Validate `actions` against the action table, and log a warning for each unknown entry dropped
- [x] 3.3 Document the `[service_commands]` section in `packaging/config.toml`

## 4. Action declaration in the daemon

- [x] 4.1 Implement feature detection: run `tedge config get service.plugin_paths`, split the paths, and look for the `container` plugin file. Resolve once at startup and log the outcome with its reason
- [x] 4.2 Resolve the hosting container once at startup with `Self()`, treating `ErrNotFound` as running on the host
- [x] 4.3 Add the `serviceActionsDeclared` set to `App`. It needs no mutex because, like `lastOrphansCheck`, it is only used from the worker goroutine. Add a `declareServiceActions(item, target)` method that publishes retained `{}` on each configured `cmd/<action>` and clears the supported actions that are not configured
- [x] 4.4 Call `declareServiceActions` from `doUpdate` after registration, skipping the hosting container (and clearing its known actions) and any service already in the set. Add a service to the set only when all its publishes succeed
- [x] 4.5 Remove a service from the set when it is deregistered in the stale removal of `doUpdate`, which is the only deregistration path
- [x] 4.6 When the feature resolves to disabled, clear the known action topics of each registered service once per process
- [x] 4.7 Add unit tests for the declaration bookkeeping: declare once, re-declare after deregistration, retry after a failed publish, skip self

## 5. Packaging

- [x] 5.1 Add the shims `packaging/service-plugins/container` and `packaging/service-plugins/container-group` (`exec tedge-container service-plugins <type> "$@"`)
- [x] 5.2 Add both shims to `.goreleaser.yaml` with destination `/usr/share/tedge/service-plugins/`, mode `0755` and owner and group `root`
- [x] 5.3 Build the packages and check that the files, ownership and modes are correct in the deb, rpm and apk packages

## 6. System tests

- [x] 6.1 Create `tests/service-commands.robot`, skipped on thin-edge.io <= 2.0.1, detected by `tedge config get service.plugin_paths` failing on the device
- [x] 6.2 Test that the container service gets `c8y_SupportedServiceCommands` = `RESTART`, `START`, `STOP` and the matching retained capability topics
- [x] 6.3 Test `c8y_ServiceCommand` `RESTART`, `STOP` and `START` on a container: the operation is SUCCESSFUL and the container state changes as expected
- [x] 6.4 Test the same for a container-group service (`<project>@<service>`)
- [x] 6.5 Test direct plugin calls: an unsupported action exits 2, and an unknown container exits 1
- [x] 6.6 Test that the declared actions are gone after the container is removed
- [x] 6.7 Test that `service_commands.enabled = false` declares no actions
- [x] 6.8 Cover rootless podman: the `podman`-tagged suite also runs in the rootless matrix job (`--include podman --exclude rootful`)
- [x] 6.9 Cover the hosting container: the test devices run their own engine (docker-in-docker), so the bundle setup cannot be reproduced. Unit tests cover skipping it (`Test_DeclareServiceActionsSkipsSelf`)

## 7. Documentation

- [x] 7.1 Add a README section on service commands: requirements, actions, configuration, the self-container limitation, and the stop-timeout note
