# Container registries

## Private container registries

Container images can be pulled from private repositories, however you must provide credentials by either adding credentials to a file or environment variables.

### Set credentials dynamically

Container registry credentials can be provided dynamically to plugin by creating an executable file called `registry-credentials` which then returns the credentials to use for the api call.


The `registry-credentials` is executed by tedge-container-plugin when it attempts to pull an image, and the image/tag is passed as an argument to the executable.

```sh
registry-credentials get IMAGE_TAG [--refresh]
```

For example:

```sh
# Get credentials for the registry related to an image
registry-credentials get ghcr.io/thin-edge/tedge:1.3.1

# Get credentials but the tedge-container-plugin has indicated that a new token
# should be issued as the last token failed
registry-credentials get ghcr.io/thin-edge/tedge:1.3.1 --refresh
```

The script should return with an exit-code 0 and the credentials in the following json format:

```json
{
   "username": "myuser",
   "password": "..."
}
```

**file: /usr/bin/registry-credentials**

```sh
#!/bin/sh
set -e
ACTION="$1"
shift

get_credentials() {
    IMAGE="$1"

    # Write log messages to stderr

    if [ "$2" = "--refresh" ]; then
        # Fetch some credentials from anywhere, e.g. api, local file storage, keychain etc.
        echo "Retrieving private repository credentials for $IMAGE" >&2
    else
        echo "Using credentials from cache (assuming they have already been cached)" >&2
    fi
    
    # Then return credentials
    cat <<EOT
{
    "username": "myuser",
    "password": "..."
}
EOT
}

case "$ACTION" in
    get)
        get_credentials "$@"
        ;;
    *)
        echo "Unknown command" >&2
        exit 1
        ;;
esac

exit 0
```

### Using static settings

Static credentials for different repositories can be provided in the following file.

**file: /data/tedge-container-plugin/credentials.toml**

```toml
[registry1]
repo = "docker.io"
username = "example"
password = ""

[registry2]
repo = "quay.io"
username = "otherUser"
password = ""
```

The file location of the `credentials.toml` file location can be changed by setting the following value in the `tedge-container-plugin.toml`:

```toml
registry.credentials_path = "/data/tedge-container-plugin/credentials.toml"
```

You can also control the same registry settings using environment variables:

```sh
CONTAINER_REGISTRY1_REPO=docker.io
CONTAINER_REGISTRY1_USERNAME=example
CONTAINER_REGISTRY1_PASSWORD=

CONTAINER_REGISTRY2_REPO=quay.io
CONTAINER_REGISTRY2_USERNAME=otherUser
CONTAINER_REGISTRY2_PASSWORD=
```
