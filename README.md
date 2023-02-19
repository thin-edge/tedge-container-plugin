# tedge-container-plugin

thin-edge.io container plugin to install, start, stop and monitor containers on a device.

## Plugin summary

The following thin-edge.io customization is included in the plugin.

:construction: :construction: :construction: Uses planned thin-edge.io features :construction: :construction: :construction:
The container monitoring requires a thin-edge.io feature (service monitoring) which is not yet merged into the `main` branch (as of `0.9.0`).

### What will be deployed to the device?

* A service called `tedge-container-plugin`. This provides the monitoring of the containers
* A software management plugin which is called when installing and removing containers via Cumulocity IoT

|Type|Yes/No|Child device|
|----|--|--|
|Software Management Plugin|✅|:x:|
|Telemetry|✅|:x:|
|Monitoring|✅|:x:|
|Operation Handler|➖|➖|

**Note**

Child device support does not make any sense with this plugin as it needs to install/remove/monitor containers running on the current device (where the container engine is running). Though I guess you could try modifying the `DOCKER_HOST` environment variable etc. Though PRs are welcome to extend/edit any of the features ;)

**Technical summary**

The following details the technical aspects of the plugin to get an idea what systems it supports.

|||
|--|--|
|**Languages**|`shell` (posix compatible)|
|**CPU Architectures**|`all/noarch`. Not CPU specific, supported everyone in single package|
|**Supported init systems**|`systemd` and `init.d/open-rc`|
|**Required Dependencies**|-|
|**Optional Dependencies (feature specific)**|`mosquitto_sub`|

### How to do I get it?

The following linux package formats are provided on the releases page and also in the [tedge-community](https://cloudsmith.io/~thinedge/repos/community/packages/) repository:

|Operating System|Repository link|
|--|--|
|Debian/Raspian (deb)|[![Latest version of 'tedge-container-plugin' @ Cloudsmith](https://api-prd.cloudsmith.io/v1/badges/version/thinedge/community/deb/tedge-container-plugin/latest/a=all;d=any-distro%252Fany-version;t=binary/?render=true&show_latest=true)](https://cloudsmith.io/~thinedge/repos/community/packages/detail/deb/tedge-container-plugin/latest/a=all;d=any-distro%252Fany-version;t=binary/)|
|Alpine Linux (apk)|[![Latest version of 'tedge-container-plugin' @ Cloudsmith](https://api-prd.cloudsmith.io/v1/badges/version/thinedge/community/alpine/tedge-container-plugin/latest/a=noarch;d=alpine%252Fany-version/?render=true&show_latest=true)](https://cloudsmith.io/~thinedge/repos/community/packages/detail/alpine/tedge-container-plugin/latest/a=noarch;d=alpine%252Fany-version/)|
|RHEL/CentOS/Fedora (rpm)|[![Latest version of 'tedge-container-plugin' @ Cloudsmith](https://api-prd.cloudsmith.io/v1/badges/version/thinedge/community/rpm/tedge-container-plugin/latest/a=noarch;d=any-distro%252Fany-version;t=binary/?render=true&show_latest=true)](https://cloudsmith.io/~thinedge/repos/community/packages/detail/rpm/tedge-container-plugin/latest/a=noarch;d=any-distro%252Fany-version;t=binary/)|
## Features

The following features are supported by the plugin:

* Install/remove containers via the Cumulocity IoT software interface
* Monitor container states (e.g. up/down) via Cumulocity IoT Services (only supported from tedge >= 0.10.0)
* Download container images via Cumulocity IoT binaries if a URL is provided
* Support for multiple container engines (docker, podman, nerdctl)


## Documentation

### Install/remove containers

Containers can be installed and removed via the Cumulocity IoT Software Management interface in the Device Management Application.

The software package is modeled so that each software name corresponds to one container instance. Upon installation of a software item, the container uses the `version` field as the source of the container image/tag which is used to create the container. The software package can include an optional `url` referring to an exported container image in the gzip (compressed tarball) format (e.g. the image that you get when running `docker save <my_image> --output <my_image>.tar.gz`).

The software package properties are also describe below:

|Property|Description|
|----|-----|
|`name`|Name of the container to create and start. There can only be one instance with this name, but this name can be anything you like. It is recommended to give it a functional name, and not a version. e.g. for a MQTT broker it could be called `mqtt-broker` (not `mosquitto`).|
|`version`|Container image and tag to be used to create the container with the `name` value. (e.g. `eclipse-mosquitto:2.0.15`). The container images usually follow the format `<image>:<tag>`, where the tag is mostly used as a version description of the image.|
|`url`|Optional url pointing to the container image in a tarball format. The file is downloaded and loaded into the container engine, prior to starting the container. The image inside the gzip **MUST** match the one given by the `version` property!|

#### Configuration

The container software management plugin can be configured with the following properties.

|Property|Value|Description|Example|
|--|--|--|--|
|`PRUNE_IMAGES`|`0` or `1`|Prune any unused images after creating/deleting the containers. This is turned off by default|
|`VALIDATE_TAR_CONTENTS`|`0` or `1`|If the image is in a tarball format, then this setting controls whether the contains of the tarball should be validated against the image name and tag provided in the `version` field of the software package. This is useful to protect against accidentally uploading the wrong binary images to the wrong software packages.|
|`CONTAINER_RUN_OPTIONS`|String. Example `"--cpus 1 --memory 64m"`|Additional command options to be used when creating/starting the containers. The options will be used by all containers|

The configuration is managed from the following file, and an example of the contents are shown below.

**File**
```sh
/etc/tedge-container-plugin/env
```

**Contents**

```sh
# container sm-plugin settings
PRUNE_IMAGES=0
VALIDATE_TAR_CONTENTS=0
CONTAINER_RUN_OPTIONS="--cpus 1 --memory 64m"
```

#### Notes/Limitations

Whilst there is support for adding custom arguments to the container creation command (e.g. `docker run`), these custom arguments would be applied to each container, and cannot be made to be container specific. For example you can't add specific volume mounting for a specific container.

In the future it would make more sense to also add support for providing a `docker-compose.yaml` file in the `url` field, which can then control all of the requirements for running the container. This would eliminate the need for the plugin to know the container specifics, as everything can be clearly defined in the `docker-compose.yaml` file. Obviously, this could open up some problems, as you might want to restrict what functionality the user is allowed to use in such a file, otherwise it could open up some isolation/security issues.

### Monitoring

The plugin also includes a service which monitors the running status of the containers and includes some runtime metrics such as memory, cpu and network io.


#### Configuration

The container software management plugin can be configured with the following properties.

|Property|Value|Description|Example|
|--|--|--|--|
|`CONTAINER_CLI_OPTIONS`|`docker podman nerdctl`|List of container cli tools to auto detect. This has no effect if `CONTAINER_CLI` has a non-empty value. The first command which is found will be used. It assumes that the device is only running one container engine at a time.|
|`CONTAINER_CLI`|`podman`|Explicitly control which container cli tool will be used. Set this if you know which cli is available on the device|
|`INTERVAL`|`60`|Interval in seconds on how often the container status/telemetry should be collected. The interval will be the minimal interval as it is the time to sleep between collections|
|`TELEMETRY`|`1` or `0`|Enable/disable the container telemetry metrics such as memory etc. Regardless of this value, the containers status will still be sent, but the measurements will not.|
|`LOG_LEVEL`|`debug`, `info`, `warn`, `error`|Service log level|
|`SERVICE_TYPE`|`container`|Service type to be used in the service monitoring.|

The configuration is managed from the following file, and an example of the contents are shown below.

**File**
```sh
/etc/tedge-container-plugin/env
```

**Contents**

```sh
CONTAINER_CLI_OPTIONS="docker podman nerdctl"
CONTAINER_CLI=docker

# Interval in seconds
INTERVAL=60

# Enable/disable telemetry (1/0)
TELEMETRY=1

# Only used if tedge cli is not installed
MQTT_HOST=127.0.0.1
MQTT_PORT=1883

# Log levels: error, warn, info, debug, none
LOG_LEVEL=info
LOG_TIMESTAMPS=1

# Service type to be used for the containers
SERVICE_TYPE=container
```


#### Troubleshooting


##### Systemd

**Start**

```sh
sudo systemctl start tedge-container-monitor
```

**Stop**

```sh
sudo systemctl stop tedge-container-monitor
```

**Reload (configuration)**

```sh
sudo systemctl reload tedge-container-monitor
```

**Get Logs**

```sh
sudo journalctl -u tedge-container-monitor -f
```

##### init.d/open-rc

**Start**

```sh
sudo service tedge-container-monitor start
```

**Stop**

```sh
sudo service tedge-container-monitor stop
```

**Reload (configuration)**

```sh
sudo service tedge-container-monitor reload
```

**Get Logs**

```sh
tail -f /var/log/tedge-container-monitor.err
```

## Developers

This section details everything you need to know about building the package yourself.

### Building

To build the linux packages use the following steps:

1. Checkout the project

2. Install [nfpm](https://nfpm.goreleaser.com/install/)

    **Note**
    Make sure you install it somewhere that is included in your `PATH` environment variable. Use `which nfpm` to check if your shell can find it after installation.

3. Build the packages

    ```sh
    ./ci/build.sh
    ```

    Ideally the `SEMVER` environment variable should be set to the git tag, however
    you can also use a manual version using:

    ```sh
    ./ci/build.sh 1.0.1
    ```

    The built packages are created under the `./dist` folder.
