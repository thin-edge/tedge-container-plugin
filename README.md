# tedge-container-plugin

thin-edge.io container management plugin which enables you to install, remove and monitor containers on a device.

There is a custom Cumulocity UI Plugin to display these container directly from the UI.

## Plugin summary

The following thin-edge.io customization is included in the plugin.

The instructions assume that you are using thin-edge.io &gt;= 1.0.0

### What will be deployed to the device?

* A service called `tedge-container-plugin`. This provides the monitoring of the containers
* The following software management plugins which is called when installing and removing containers/container groups via Cumulocity IoT
    * `container` - Deploy a single container (`docker run xxx` equivalent)
    * `container-group` - Deploy one or more container as defined by a `docker-compose.yaml` file (`docker compose up` equivalent), or an archive (gzip or zip)


**Technical summary**

The following details the technical aspects of the plugin to get an idea what systems it supports.

|||
|--|--|
|**Languages**|`golang`|
|**CPU Architectures**|`armv6 (armhf)`, `armv7 (armhf)`, `arm64 (aarch64)`, `amd64 (x86_64)`|
|**Supported init systems**|`systemd` and `init.d/open-rc`|
|**Required Dependencies**|-|

### How to do I get it?

The following linux package formats are provided on the releases page and also in the [tedge-community](https://cloudsmith.io/~thinedge/repos/community/packages/) repository:

|Operating System|Repository link|
|--|--|
|Debian/Raspbian (deb)|[![Latest version of 'tedge-container-plugin-ng' @ Cloudsmith](https://api-prd.cloudsmith.io/v1/badges/version/thinedge/community/deb/tedge-container-plugin-ng/latest/a=arm64;d=any-distro%252Fany-version;t=binary/?render=true&show_latest=true)](https://cloudsmith.io/~thinedge/repos/community/packages/detail/deb/tedge-container-plugin-ng/latest/a=arm64;d=any-distro%252Fany-version;t=binary/)|
|Alpine Linux (apk)|[![Latest version of 'tedge-container-plugin-ng' @ Cloudsmith](https://api-prd.cloudsmith.io/v1/badges/version/thinedge/community/alpine/tedge-container-plugin-ng/latest/a=aarch64;d=alpine%252Fany-version/?render=true&show_latest=true)](https://cloudsmith.io/~thinedge/repos/community/packages/detail/alpine/tedge-container-plugin-ng/latest/a=aarch64;d=alpine%252Fany-version/)|
|RHEL/CentOS/Fedora (rpm)|[![Latest version of 'tedge-container-plugin-ng' @ Cloudsmith](https://api-prd.cloudsmith.io/v1/badges/version/thinedge/community/alpine/tedge-container-plugin-ng/latest/a=aarch64;d=alpine%252Fany-version/?render=true&show_latest=true)](https://cloudsmith.io/~thinedge/repos/community/packages/detail/alpine/tedge-container-plugin-ng/latest/a=aarch64;d=alpine%252Fany-version/)|
## Features

The following features are supported by the plugin:

* Install/remove containers via the Cumulocity IoT software interface
* Install multiple containers as one group using a `docker-compose.yaml` file or an archive container a `docker-compose.yaml` file
* Monitor container states (e.g. up/down) via Cumulocity IoT Services (only supported from tedge >= 1.0.0)
* Download container images via Cumulocity IoT binaries if a URL is provided
* Support for multiple container engines (docker, podman)


## Documentation

### Install/remove single containers

Containers can be installed and removed via the Cumulocity IoT Software Management interface in the Device Management Application.

The software package is modeled so that each software name corresponds to one container instance. Upon installation of a software item, the container uses the `version` field as the source of the container image/tag which is used to create the container. The software package can include an optional `url` referring to an exported container image in the gzip (compressed tarball) format (e.g. the image that you get when running `docker save <my_image> --output <my_image>.tar.gz`).

The software package properties are also describe below:

|Property|Description|
|----|-----|
|`name`|Name of the container to create and start. There can only be one instance with this name, but this name can be anything you like. It is recommended to give it a functional name, and not a version. e.g. for a MQTT broker it could be called `mqtt-broker` (not `mosquitto`).|
|`version`|Container image and tag to be used to create the container with the `name` value. (e.g. `eclipse-mosquitto:2.0.15`). The container images usually follow the format `<image>:<tag>`, where the tag is mostly used as a version description of the image|
|`softwareType`|`container`. This indicates that the package should be managed by the `container` software management plugin|
|`url`|Optional url pointing to the container image in a tarball format. The file is downloaded and loaded into the container engine, prior to starting the container. The image inside the gzip **MUST** match the one given by the `version` property!|

#### Private container registries

Pulling image from private container registries is supported. Check out the [container registries](./docs/CONTAINER_REGISTRIES.md) documentation for the available options.

### Install/remove a `container-group`

A `container-group` is the name given to deploy a `docker-compose.yaml` file or an archive (zip or gzip file) with the `docker-compose.yaml` file at the root level of the archive. A docker compose file allows use to deploy multiple containers/networks/volumes and allows you maximum control over how the container is started. This means you can create a complex setup of persisted volumes, isolated networks, and also facilitate communication between containers. Check out the [docker compose documentation](https://docs.docker.com/compose/compose-file/) for more details on how to write your own service definition.

The software package properties are also describe below:

|Property|Description|
|----|-----|
|`name`|Name of the project (this will be the logical name that represents all of the services/networks/volumes in the docker compose file|
|`version`|A custom defined version number to help track which version of the docker compose file is deployed. Technically this can be anything as it does not have an influence on the actual docker compose command, it is purely used for tracking on the cloud side|
|`softwareType`|`container-group`. This indicates that the package should be managed by the `container-group` software management plugin|
|`url`|The url to the uploaded `docker-compose.yaml` file. This is a MANDATORY field and cannot be left blank.|


### Monitoring

The plugin also includes a service which monitors the running status of the containers and includes some runtime metrics such as memory, cpu and network io. Please note that access to the container monitoring might not be supported by your container engine. When in doubt, just manually do a `docker stats` and if the data is only showing zeros, then the plugin will also see zeros.

#### Telemetry

Checkout the [TELEMETRY](./docs/TELEMETRY.md) docs for details on what is included in the telemetry data.


#### Configuration

The tedge-container-plugin can be configured with the following properties.

A default configuration is provided in the package, [tedge-container-plugin/config.toml](./packaging/config.toml) and it include a description of each property.


**Default Configuration file location**

```sh
/etc/tedge-container-plugin/config.toml
```

**Note**

The configuration can be controlled by setting environment variables. The configuration property name to environment variable name can be translated using the following rules:

* Use the `CONTAINER_` prefix
* Upper case the 
* Replace the `.` character with an underscore `_`

Below are some examples showing the mapping between the configuration values and environment variables:

|Configuration|EnvironmentVariable|
|-------------|-------------------|
|`filter.exclude.name = ["type1", "type2"]`| `CONTAINER_FILTER_EXCLUDE_NAME=type1,type2` |
|`container.alwayspull= true`| `CONTAINER_CONTAINER_ALWAYSPULL=true` |
|`container.network= true`| `CONTAINER_CONTAINER_NETWORK=tedge` |

### UI Plugin

With the UI plugin, container monitoring can be added to the Cumulocity UI. The Ui plugin contains 3 components that add the following tabs to the UI:

| Plug-In                        | Function                                                                                                                   |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| Container Info Tab             | Adds a tab to a container service to display all relevant container information.                                           |
| Container Management Tab       | Adds a tab to the device to monitor containers. The tab can include/exclude the containers hosted within container groups. |
| Container Group Management Tab | Adds a tab to the device to monitor container groups (aka. docker compose).                                                |

The UI Plugin was developed for the Device Management application, but can also be added to the Cockpit if needed.

To use the UI-Plugin, download the tedge-container-plugin-ui.zip version that matches your Cumulocity UI version form the releases and upload it in the Cumulocity administration. Detailed instructions on how to install UI plugins can also be found [here](https://cumulocity.com/guides/users-guide/administration/#custom-applications).

#### Container Info Tab

The tab will be enabled for all services of type container. Displays the container properties that are stored in the managed Object.
![Container Info Screenshot](./docs/img/container-info.png)

#### Container Management Tab

The tab will be enabled for all devices with a childAddition with serviceType=container. Lists all containers in a grid or list.The search can be used for the image name and the project id. The list can include/exclude the containers that are part of a container group.
![Container Container Management Screenshot](./docs/img/container-management.png)

#### Container Group Management Tab

The tab will be enabled for all devices with a childAddition with serviceType=container. Lists all containers that are part of a project. The filter/search can be used to search for project names or container images.
![Container Container Management Screenshot](./docs/img/container-group-management.png)

## Developers

This section details everything you need to know about building the package yourself.

### Building

To build the project use the following steps:

1. Checkout the project

2. Install [goreleaser]https://goreleaser.com/install/)

    **Note**

    Make sure you install it somewhere that is included in your `PATH` environment variable. Use `which goreleaser` to check if your shell can find it after installation.

3. Build the packages

    ```sh
    just release-local
    ```

    The built packages are created under the `./dist` folder.

### Running system tests

You can run the system tests can be run locally, however if you're having problem, look at the [test.yaml](.github/workflows/test.yaml) workflow for the tests as this is known to work.

If you're using VS Code, then you can also install the following extensions to enable running tests via the `tests/*robot` files:

* robocorp.robocorp-code
* robocorp.robotframework-lsp

To run the tests you will need to have python3 &gt;> 3.9 installed on your system, then run the following

1. Create an initial `.env` file and fill in your Cumulocity credentials to be used for the tests

   ```sh
   just init-dotenv
   ```

2. Build the software management plugin

   ```sh
   just release-local
   ```

3. Build the test images

   ```sh
   just build-test
   ```

4. Setup the python3 virtual environment and install the test dependencies

   ```sh
   just venv
   ```

5. Run the RobotFramework tests

   ```sh
   just test --include podman
   just test --include docker
   ```

### Building UI

To build the ui use the following steps:

1. Checkout the project

2. Install dependencies:

   ```sh
   npm install
   ```

3. (Optional) Run the UI locally.

   Add your tenant in the package.json file:

   ```json
    "scripts": {
    "start": "c8ycli server -u https://{{add the url of your tenant here}} --shell devicemanagement",
    ...
    }
   ```

   Start the UI locally via:

   ```sh
   npm start
   ```

4. Build the Plugin

   ```sh
    npm run build
   ```

5. Deploy the Plugin

   ```sh
   npm run deploy
   ```
