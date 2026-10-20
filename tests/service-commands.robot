*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Test Setup    Test Setup
Test Teardown    Collect Logs

Test Tags    docker    podman

*** Variables ***
${DEVICE_SN}    ${EMPTY}

*** Test Cases ***

Container service declares the supported service commands
    Create Container    svc-app1
    Service Should Have Service Commands    svc-app1    ["RESTART","START","STOP"]
    Service Should Have Retained Actions    svc-app1    restart    start    stop

Restart, stop and start a container from Cumulocity
    Create Container    svc-app1
    Service Should Have Service Commands    svc-app1    ["RESTART","START","STOP"]

    ${started_at_before}=    Get Container Start Time    svc-app1
    ${operation}=    Create Service Command Operation    svc-app1    {"command":"RESTART","serviceName":"svc-app1","serviceType":"container"}
    Cumulocity.Operation Should Be SUCCESSFUL    ${operation}    timeout=120
    ${started_at_after}=    Get Container Start Time    svc-app1
    Should Not Be Equal As Strings    ${started_at_after}    ${started_at_before}

    ${operation}=    Create Service Command Operation    svc-app1    {"command":"STOP"}
    Cumulocity.Operation Should Be SUCCESSFUL    ${operation}    timeout=120
    Container Should Have State    svc-app1    exited
    Cumulocity.Should Have Services    name=svc-app1    service_type=container    status=down

    ${operation}=    Create Service Command Operation    svc-app1    {"command":"START"}
    Cumulocity.Operation Should Be SUCCESSFUL    ${operation}    timeout=120
    Container Should Have State    svc-app1    running
    Cumulocity.Should Have Services    name=svc-app1    service_type=container    status=up

Restart a container-group service from Cumulocity
    ${binary_url}=    Cumulocity.Create Inventory Binary    nginx    container-group    file=${CURDIR}/data/docker-compose.nginx.yaml
    ${operation}=    Cumulocity.Install Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group", "url": "${binary_url}"}
    Cumulocity.Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Cumulocity.Should Have Services    name=nginx@nginx    service_type=container-group    status=up
    Service Should Have Service Commands    nginx@nginx    ["RESTART","START","STOP"]

    ${container_id}=    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker ps -aq --filter label=com.docker.compose.project=nginx --filter label=com.docker.compose.service=nginx    strip=${True}
    ${started_at_before}=    Get Container Start Time    ${container_id}
    ${operation}=    Create Service Command Operation    nginx@nginx    {"command":"RESTART"}
    Cumulocity.Operation Should Be SUCCESSFUL    ${operation}    timeout=120
    ${started_at_after}=    Get Container Start Time    ${container_id}
    Should Not Be Equal As Strings    ${started_at_after}    ${started_at_before}

    ${operation}=    Create Service Command Operation    nginx@nginx    {"command":"STOP"}
    Cumulocity.Operation Should Be SUCCESSFUL    ${operation}    timeout=120
    Container Should Have State    ${container_id}    exited

    ${operation}=    Create Service Command Operation    nginx@nginx    {"command":"START"}
    Cumulocity.Operation Should Be SUCCESSFUL    ${operation}    timeout=120
    Container Should Have State    ${container_id}    running

Service plugin exit codes
    Create Container    svc-app1
    # Unsupported action
    DeviceLibrary.Execute Command    cmd=sudo tedge service enable svc-app1 --type container    exp_exit_code=2
    DeviceLibrary.Execute Command    cmd=sudo tedge-container service-plugins container enable svc-app1    exp_exit_code=2
    # Unknown service
    DeviceLibrary.Execute Command    cmd=sudo tedge-container service-plugins container restart does-not-exist    exp_exit_code=1
    DeviceLibrary.Execute Command    cmd=sudo tedge-container service-plugins container-group restart does-not-exist@web    exp_exit_code=1
    DeviceLibrary.Execute Command    cmd=sudo tedge-container service-plugins container-group restart invalid-name    exp_exit_code=1
    # Supported, but not declarable action
    DeviceLibrary.Execute Command    cmd=sudo tedge service pause svc-app1 --type container
    Container Should Have State    svc-app1    paused
    DeviceLibrary.Execute Command    cmd=sudo tedge service unpause svc-app1 --type container
    Container Should Have State    svc-app1    running

Removed container leaves no declared actions
    Create Container    svc-app2
    Service Should Have Retained Actions    svc-app2    restart    start    stop
    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools container-remove svc-app2
    Cumulocity.Should Have Services    name=svc-app2    service_type=container    min_count=0    max_count=0
    Service Should Have Retained Actions    svc-app2

Service commands can be disabled
    DeviceLibrary.Execute Command    cmd=echo "CONTAINER_SERVICE_COMMANDS_ENABLED=false" | sudo tee -a /etc/tedge-container-plugin/env
    DeviceLibrary.Restart Service    tedge-container-plugin
    Create Container    svc-app3
    Cumulocity.Should Have Services    name=svc-app3    service_type=container    status=up
    Service Should Have Retained Actions    svc-app3

*** Keywords ***

Test Setup
    ${DEVICE_SN}=    Setup
    Set Test Variable    $DEVICE_SN
    Cumulocity.External Identity Should Exist    ${DEVICE_SN}
    Cumulocity.Should Have Services    name=tedge-container-plugin    service_type=service    min_count=1    max_count=1    timeout=30

    # The service commands require thin-edge.io > 2.0.1 (currently only the main channel).
    # Detect it by the service.plugin_paths setting (like the plugin's "auto" mode) rather
    # than by version number, so the tests are enabled once a release includes the feature
    ${output}=    DeviceLibrary.Execute Command    cmd=tedge config get service.plugin_paths >/dev/null 2>&1 && echo supported || echo unsupported    strip=${True}
    Skip If    "${output}" != "supported"    Service commands require thin-edge.io > 2.0.1 with service plugin support

    # Create common network for all containers
    ${operation}=    Cumulocity.Execute Shell Command    set -a; . /etc/tedge-container-plugin/env; docker network create tedge ||:

    # Create data directory
    DeviceLibrary.Execute Command    mkdir -p /data

Create Container
    [Arguments]    ${name}
    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools container-remove ${name} ||: ; sudo tedge-container engine docker run -d --network bridge --name ${name} ghcr.io/thin-edge/test-images/httpd:2.4.61-alpine
    Cumulocity.Should Have Services    name=${name}    service_type=container    status=up

Service Should Have Service Commands
    [Arguments]    ${service}    ${commands}
    Cumulocity.External Identity Should Exist    ${DEVICE_SN}:device:main:service:${service}
    Cumulocity.Managed Object Should Have Fragment Values    c8y_SupportedServiceCommands\=${commands}    timeout=30
    # Restore the device as the current managed object
    Cumulocity.External Identity Should Exist    ${DEVICE_SN}

Service Should Have Retained Actions
    [Documentation]    Check the retained cmd/<action> capabilities of a service. No actions means none are expected
    [Arguments]    ${service}    @{actions}
    ${output}=    DeviceLibrary.Execute Command    cmd=tedge mqtt sub 'te/device/main/service/${service}/cmd/+' --retained-only --duration 2s | grep -o 'cmd/[a-z_-]*' | sed 's|cmd/||' | sort | tr '\\n' ' '    strip=${True}    ignore_exit_code=${True}
    ${expected}=    Evaluate    " ".join(sorted($actions))
    Should Be Equal As Strings    ${output}    ${expected}

Create Service Command Operation
    [Arguments]    ${service}    ${fragment}
    Cumulocity.External Identity Should Exist    ${DEVICE_SN}:device:main:service:${service}
    ${operation}=    Cumulocity.Create Operation    fragments={"c8y_ServiceCommand":${fragment}}    description=Service command
    # Restore the device as the current managed object
    Cumulocity.External Identity Should Exist    ${DEVICE_SN}
    RETURN    ${operation}

Get Container Start Time
    [Arguments]    ${name}
    ${started_at}=    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker container inspect ${name} --format "{{ .State.StartedAt }}"    strip=${True}
    RETURN    ${started_at}

Container Should Have State
    [Arguments]    ${name}    ${state}
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker container inspect ${name} --format "{{ .State.Status }}" | grep -x ${state}    retries=10
