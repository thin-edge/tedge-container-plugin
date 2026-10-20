## Context

thin-edge.io's service commands run end to end without anything from this repo, except the service plugin that does the work and the declaration of the supported actions:

```
Cumulocity c8y_ServiceCommand
  → tedge-mapper-c8y   (accepts only actions present in c8y_SupportedServiceCommands)
  → te/<service>/cmd/<action>/<id> {serviceName, serviceType}
  → tedge-agent shipped workflow service_commands.toml (operation = start|stop|restart|enable|disable, type = "service")
  → sudo -n tedge service <action> <serviceName> --type <serviceType>   (180 s timeout)
  → <service.plugin_paths>/<serviceType> <action> <serviceName>          ← this change
```

Current state of this repo:
- `doUpdate` (`pkg/app/app.go`) registers one service for each container through the entity HTTP API:
  - the type is `container` or `container-group`
  - the name is the container name, or `<project>@<service>` (`Container.GetName`, subject to `container_group.use_module_name`)
- `List` uses `All: true`, so stopped containers stay registered. A `stop` therefore does not remove the service, and a later `start` can still find it.
- Deregistration goes through `DeleteEntity`. tedge-agent's `deregister_entity` already clears the `AnyCommandMetadata` retained topics.
- Log plugins follow a pattern that can be reused:
  - shell shims in `packaging/log-plugins/` that `exec tedge-container log-plugins <type> "$@"`
  - the `cli/log_plugins/{container_plugin,container_group_plugin}` packages
  - the container-group name resolution (`ParseContainerGroup` → `ResolveComposeProjectName` → `LookupProject`)
- `ContainerClient` already has `StartContainer`, `StopContainer` and `RestartContainer`. It has no pause or unpause yet.
- `ContainerClient.Self()` finds the container the process runs in. The daemon does not use it today.

Constraints from the Service Plugin API:
- argv-based invocation `<plugin> <action> <name>`
- exit codes `0` (success), `2` (unsupported) and any other non-zero code (failure, reason on stderr)
- the plugin runs as root, from a directory owned by root that the `tedge` user cannot write to
- the output must not contain `:::begin-tedge:::` / `:::end-tedge:::`

## Goals / Non-Goals

**Goals:**
- Start, stop and restart `container` and `container-group` services from Cumulocity, with no custom workflow.
- `pause` and `unpause` in the service plugin, for direct use and for custom workflows. They are not declared by the daemon, because tedge-agent ships no workflow for them.
- Declare actions only when they will work: thin-edge.io supports service plugins, the plugin is installed, and the target is not the plugin's own container.
- Degrade gracefully on older thin-edge.io: no declared actions, and behavior is otherwise unchanged.

**Non-Goals:**
- `enable` and `disable`. Mapping them to the restart policy is a possible follow-up. For now the plugin exits `2`.
- Actions on a whole compose project. Each service entity is one compose service container.
- Actions on the plugin's own container. A safe self-restart through a forked helper container is a possible follow-up.
- Changes to tedge-agent or to the shipped workflow.

## Decisions

### D1. Plugin entry points: subcommands behind shell shims
`tedge-container service-plugins container|container-group <action> <name>`, launched by `/usr/share/tedge/service-plugins/{container,container-group}`, which contain `#!/bin/sh` plus `exec tedge-container service-plugins <type> "$@"`.
- *Alternative*: a symlink to the binary with dispatch on argv[0], like the sm-plugins. Rejected: argv[0] `container` already selects the software-management plugin, so the two plugin kinds could not be told apart.
- *Alternative*: a static `case` shell script calling the docker CLI, as in the upstream example. Rejected: it would not handle podman or the API socket configuration, nor the `<project>@<service>` resolution.

### D2. Actions are a cobra positional argument, not one subcommand each
`service-plugins container` takes exactly 2 args, `<action> <name>`, and dispatches on a table (`map[string]func(ctx, id) error`). An unknown action returns a sentinel `ErrUnsupportedAction`. `main` maps that error to exit code `2` and every other error to `1`. This keeps exit code 2 out of cobra's own "unknown command" handling and gives one place to own the contract.
- The name is passed through unchanged.
- A name starting with `-` cannot reach the plugin, because `tedge service` rejects it first. Even so, the plugin uses `cobra.ExactArgs(2)` and sets `DisableFlagParsing` on these leaf commands, so a value is never read as a flag.

### D3. Engine operations
| Action    | Engine call                              |
|-----------|------------------------------------------|
| start     | `ContainerStart`                         |
| stop      | `ContainerStop` (container's own stop timeout) |
| restart   | `ContainerRestart`                       |
| pause     | `ContainerPause` (new client method)     |
| unpause   | `ContainerUnpause` (new client method)   |

Calls are idempotent where the engine allows it. Starting a running container and stopping a stopped one both succeed: Docker returns 304, which the SDK treats as success. Podman's compat API needs checking in tests. Health and status are not published by the plugin. The daemon's engine-event monitor already reacts to the state change and keeps one source of truth.

### D4. Name resolution
- `container`: `ContainerInspect(name)`. A missing container exits `1` with "container not found".
- `container-group`: a shared helper `ContainerClient.LookupGroupService(ctx, name)`, also used by the log plugin, so both stay consistent under `use_module_name`. It:
  - parses `<project>@<service>`
  - resolves the module name to the compose project
  - lists the containers, including stopped ones, matching the exact `com.docker.compose.project` and `com.docker.compose.service` labels

  The old `LookupProject` listed only running containers and matched the compose service as a substring of the container name, so `start` could not find a stopped service.
- The service plugin requires exactly one match. No match exits `1`, and so does more than one match: scaled services are ambiguous, so the error lists the matches instead of picking one. The log plugin keeps using the first match.

### D5. Declaring actions from the daemon
After the registration loop in `doUpdate`, for each registered service not yet in `a.declaredActions` (a set of topic IDs held in memory and guarded by a mutex), publish retained `{}` on `te/<topic-id>/cmd/<action>` for each configured action, at QoS 1. The topic ID is added to the set only once all publishes succeed.
- Remove a topic ID from the set when the service is deregistered, so a re-created container is declared again.
- The set starts empty at process start, so actions are re-declared once after a restart. That is idempotent, and the mapper publishes nothing for an action that is already known.
- The configured action list can change between restarts. On first declaration the plugin also clears (empty retained payload) the declarable actions (`start, stop, restart`) that are not configured, so narrowing the list takes effect. Other actions, e.g. a `pause` declared manually alongside a custom workflow, are never touched.
- Actions are declared only for services whose registration succeeded in the same cycle, and after the registration and twin loops. A capability of an unknown service could auto-register it with the default type `service`.
- *Alternative*: declare on every `doUpdate`. Rejected: 3–5 retained publishes per container per cycle, with event-driven updates as well.
- *Alternative*: read back retained topics to diff. Rejected: it needs a subscribe-and-wait round trip for little gain.

### D6. Feature detection (`service_commands.enabled = "auto"`)
At daemon start, when `enabled` is `auto` (the default), resolve support once:
1. Run `tedge config get service.plugin_paths`. A non-zero exit, or `tedge` missing, means unsupported.
2. Parse the value, which `tedge config get` prints as a JSON-like list (e.g. `["/usr/share/tedge/service-plugins"]`), falling back to a comma-separated list. Support requires one of the directories to contain the `container` plugin file.

With `true`, the check is skipped: this covers a plugin running somewhere without the `tedge` CLI next to it. With `false`, nothing is declared. The outcome is logged at info level, along with the reason. The check is not re-run until the next start: a thin-edge.io upgrade restarts services through packaging anyway.
- *Alternative*: compare `tedge --version`. Rejected: brittle against pre-releases and backports. Probing the setting matches the approach in commit 6b77120.

### D7. Self-protection
At start, call `Self()` once. `ErrNotFound` means the plugin runs on the host, and no container is excluded.
- When the plugin runs in a container, that container's ID and name are stored. Its service gets no declared actions, and any previously declared action is cleared.
- The service plugin itself also calls `Self()` and refuses every action on the matching container with exit `1`: "refusing to <action> the container hosting thin-edge.io". This keeps a direct `tedge service restart` or a stale declaration from killing the agent mid-workflow. Without the refusal, tedge-agent would resume at `execute` and loop.
- Matching is on the container ID, not the name, after resolution.

### D8. Configuration
```toml
[service_commands]
# "auto": declare actions only when thin-edge.io supports service plugins
# true: always declare, false: never declare
enabled = "auto"
# Actions declared on each container / container-group service: start, stop, restart
actions = ["start", "stop", "restart"]
```
- Entries in `actions` that cannot be declared are logged at warn level and dropped. That covers unknown actions, and `pause` and `unpause`, which have no shipped workflow.
- The service plugin itself does not consult `actions`. It runs anything in its table. `actions` controls only what the cloud is offered.
- Like every other setting, both can be set through the environment, e.g. `CONTAINER_SERVICE_COMMANDS_ENABLED=false`.

### D9. Packaging
The goreleaser `contents` entries add both shims to `/usr/share/tedge/service-plugins/`, with `file_info: {mode: 0755, owner: root, group: root}`. They are not `type: config`. The directory is created by thin-edge.io's own packaging, and nfpm creates it if it is missing; both give root:root `0755`. No sudoers change is needed: `tedge service` is already allowed.

## Risks / Trade-offs

- **Rootless podman under sudo**: the plugin runs as root and may not find the rootless socket that the daemon uses (its environment is different). → It reads `container.host` from `/etc/tedge/plugins/tedge-container-plugin.toml` as the other plugins do. The `podman-rootless.robot` suite gets a service-command case, and a configured `container.host` is documented as required.
- **Stop timeout vs workflow timeout**: `restart` = stop + start. A container with a long `stop_grace_period` could exceed the workflow's 180 s. → Use the container's own timeout and document the limit. With the default 10 s timeouts this is not a concern.
- **`Self()` false negative** (e.g. podman without a matching hostname): the plugin's container gets actions declared. → The plugin-side refusal (D7) still prevents harm. The cost is an operation that fails visibly.
- **Feature probe at start only**: if thin-edge.io is upgraded without restarting the plugin, no actions are declared until the next restart. → Acceptable, and documented.
- **`Self()` false positive or negative**: on the host, `IsInsideContainer()` is false, so no container is excluded. Inside a container, the existing `Self()` falls back to matching `Config.Hostname`, and containers with `network_mode: host` share the host's hostname, so the wrong container can be matched. → This limitation is shared with self-update. A wrong match only removes actions from one container; it is not a safety problem by itself, but the real hosting container is then not protected.
- **Broker loses retained messages** (mosquitto without persistence restarted): the declarations are gone, but the daemon considers them published until it restarts. → Accepted for now. A follow-up could re-declare on MQTT reconnect.
- **Retained declarations outlive a disabled feature**: changing `enabled` to `false` leaves the existing `cmd/*` topics. → When the feature resolves to disabled, the plugin clears its known action topics on the services it registers, once per process. This mirrors the narrowing in D5.

## Migration Plan

- No breaking change. Upgrading the package installs the shims, and the restarted daemon declares the actions; Cumulocity shows the Commands tab with no further step.
- Rollback: downgrade the package. Stale `cmd/*` topics remain until the services are re-registered, and operations on them fail with "not supported" (exit 2 → `FAILED`). Alternatively, set `enabled = false` before downgrading to clear them.

## Open Questions

- Should `enable` / `disable` map to the restart policy (`unless-stopped` / `no`) in a follow-up?
- Is a safe self-restart (a forked helper container plus a workflow step that does not re-run on resume) worth a follow-up change?
- Does podman's compat API return success for start-when-running and stop-when-stopped, like Docker? This is to be confirmed in tests (D3).
- Should the daemon install a workflow for `pause` / `unpause`, only when service plugin support is detected? For now they are left to users, who add the workflow and declare the actions themselves.
