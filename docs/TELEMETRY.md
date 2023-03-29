# Telemetry

## Measurements

### Containers

When telemetry is activated, each container will provided the following measurements

```json
{
  "container": {
    "cpu": {
      "value": 0
    },
    "memory": {
      "value": 0
    },
    "netio": {
      "value": 0
    }
  },
  "id": "976490",
  "self": "https://example.c8y.io/measurement/measurements/976490",
  "source": {
    "id": "37690075",
  },
  "time": "2023-03-29T19:13:25.693Z",
  "type": "ThinEdgeMeasurement"
}
```

|Name|Unit|Description|
|----|----|-----------|
|`container.cpu`|Percent|the percentage of the host’s CPU the container is using|
|`container.memory`|Percent|the percentage of the host’s memory the container is using|
|`container.netio`|Bytes)|The amount of data the container has received and sent over its network interface|

## Meta information

### Containers

The following properties are stored on the service managed object.

```json
{
  "command": "\"/bin/bash /entrypoint.sh /bin/sh -c 'exec /usr/local/bin/step-ca --password-file $PWDPATH $CONFIGPATH'\"",
  "containerId": "efe670e1c55434677ff23245199dac9a2797c45c0a3ad2f4640c16e648b5adf1",
  "createdAt": "2023-03-04 16:57:26 +0100 CET",
  "filesystem": "0B (virtual 136MB)",
  "image": "smallstep/step-ca",
  "name": "gallant_liskov",
  "networks": "bridge",
  "ports": "",
  "runningFor": "3 weeks ago",
  "serviceType": "container",
  "state": "exited",
  "status": "Exited (0) 2 weeks ago",
  "type": "c8y_Service"
}
```
