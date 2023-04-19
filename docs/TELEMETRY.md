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
  "containerStatus": "Exited (0) 2 weeks ago",
  "createdAt": "2023-03-04 16:57:26 +0100 CET",
  "filesystem": "0B (virtual 136MB)",
  "image": "smallstep/step-ca",
  "name": "gallant_liskov",
  "networks": "bridge",
  "ports": "",
  "runningFor": "3 weeks ago",
  "serviceType": "container",
  "state": "exited",
  "status": "up",
  "type": "c8y_Service"
}
```

### Container Groups (container-group)

The following properties are stored on the service managed object.

```json
{
  "command": "\"/lib/systemd/systemd\"",
  "containerId": "33870eaa5f67ef7ef98b1e526a7f1956101ae586d4e439b1bc75af7986a549ba",
  "containerName": "tedge-device-tedge-1",
  "containerStatus": "Up 52 minutes",
  "createdAt": "2023-04-11 13:54:42 +0200 CEST",
  "filesystem": "8.4MB (virtual 136MB)",
  "id": "921038541",
  "image": "reubenmiller/tedge-device:0.9.0-218-gd8bd3b33-9",
  "name": "tedge-device::tedge",
  "networks": "tedge-device_default",
  "owner": "device_rmi_raspberrypi3",
  "ports": "",
  "projectName": "tedge-device",
  "runningFor": "52 minutes ago",
  "serviceName": "tedge",
  "serviceType": "container-group",
  "state": "running",
  "status": "up",
  "type": "c8y_Service"
}
```
