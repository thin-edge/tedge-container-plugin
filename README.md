# tedge-container-plugin

thin-edge.io container plugin to install, start, stop and monitor containers on a device.

## Plugin description

The following thin-edge.io customization is included in the plugin.

* Monitor container status (up/down)
* Publish container metrics such as cpu, memory, network i/o

|Type|Yes/No|
|----|--|
|Software Management|✅|
|Telemetry|✅|
|Monitoring|✅|
|Operations|➖|

## Features

The following features are supported by the plugin:

* Manage containers via the Cumulocity IoT software interface
* Monitor container states (e.g. up/down) via Cumulocity IoT Services (only supported from tedge >= 0.10.0)
* Download container images via Cumulocity IoT binaries if a URL is provided
* Support for multiple container engines (docker, podman, nerdctl)

## Packaging

The following linux package formats are provided on the releases page:

* deb (Debian/Raspbian)
* apk (Alpine Linux)
* rpm (RHEL/CentOS/Fedora)

## Building

To build the linux packages use the following steps:

1. Checkout the project

2. Install [nfpm](https://nfpm.goreleaser.com/install/)

3. Build the packages

    ```sh
    ./ci/build.sh
    ```

    Ideally the `SEMVER` environment variable should be set to the git tag, however
    you can also use a manual version using:

    ```sh
    ./ci/build.sh 1.0.1
    ```
