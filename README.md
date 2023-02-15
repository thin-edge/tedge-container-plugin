# tedge-docker-sm-plugin

thin-edge.io software management plugin to install docker images to a device.

## Packaging

The following linux package formats are provided on the releases package:

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
