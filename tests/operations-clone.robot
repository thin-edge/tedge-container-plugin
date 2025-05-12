*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Suite Setup    Suite Setup
Test Teardown    Collect Logs

Test Tags    docker    podman

*** Test Cases ***

Check for Update
    Create Container    app1    docker.io/library/httpd:2.4.61-alpine
    DeviceLibrary.Execute Command    cmd=tedge-container engine docker container inspect app1 --format "{{.Id}}"

    # No update
    DeviceLibrary.Execute Command    sudo tedge-container tools container-clone --container app1 --image httpd:2.4.61-alpine --check    exp_exit_code=2

    # Force update
    DeviceLibrary.Execute Command    sudo tedge-container tools container-clone --container app1 --image httpd:2.4.61-alpine --check --force    exp_exit_code=0

    # Update is required as local image is not available
    DeviceLibrary.Execute Command    sudo tedge-container tools container-clone --container app1 --image httpd:2.4.62-alpine --check    exp_exit_code=0

Clone Existing Container
    Create Container    app2    docker.io/library/httpd:2.4
    ${prev_container_id}=    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker container inspect app2 --format "{{.Id}}"

    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools container-clone --container app2 --force --label io.thin-edge.last_id=${prev_container_id}

    ${next_container_id}=    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker inspect app2 --format "{{.Id}}"
    Should Not Be Equal    ${next_container_id}    ${prev_container_id}

    ${label_prev_id}=    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker inspect app2 --format '{{ index .Config.Labels "io.thin-edge.last_id" }}'
    Should Be Equal    ${label_prev_id}    ${prev_container_id}

Clone Existing Container by Timeout Whilst Waiting For Exit
    Create Container    app3    docker.io/library/httpd:2.4
    ${prev_container_id}=    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker container inspect app3 --format "{{.Id}}"

    DeviceLibrary.Execute Command    sudo tedge-container tools container-clone --container app3 --force --wait-for-exit --stop-timeout 15s    exp_exit_code=!0

    ${next_container_id}=    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker inspect app3 --format "{{.Id}}"
    Should Be Equal    ${next_container_id}    ${prev_container_id}

Clone Existing Container but Waiting For Exit
    Create Container    app4    docker.io/library/httpd:2.4
    ${prev_container_id}=    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker container inspect app4 --format "{{.Id}}"

    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container tools container-clone --container app4 --force --wait-for-exit --stop-timeout 60s 2>&1

    Sleep    5s
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker container stop app4

    Cumulocity.Operation Should Be SUCCESSFUL    ${operation}
    Log    ${operation.to_json()["c8y_Command"]["result"]}

    ${next_container_id}=    DeviceLibrary.Execute Command    cmd=tedge-container engine docker inspect app4 --format "{{.Id}}"
    Should Not Be Equal    ${next_container_id}    ${prev_container_id}

Ignore Containers With Given Label
    # ignore using label
    ${prev_container_id}=    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker run -d --label tedge.ignore=1 --name httpapp1 httpd:2.4.61-alpine
    Sleep    10s
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker inspect httpapp1
    Cumulocity.Should Have Services    service_type=container    name=httpapp1    min_count=0    max_count=0    timeout=10

    # don't ignore
    ${prev_container_id}=    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker run -d --name httpapp2 httpd:2.4.61-alpine
    Sleep    10s
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker inspect httpapp2
    Cumulocity.Should Have Services    service_type=container    name=httpapp2    min_count=1    max_count=1    timeout=10

*** Keywords ***

Suite Setup
    ${DEVICE_SN}=    Setup
    Set Suite Variable    $DEVICE_SN
    Cumulocity.External Identity Should Exist    ${DEVICE_SN}
    Cumulocity.Should Have Services    name=tedge-container-plugin    service_type=service    min_count=1    max_count=1    timeout=30

    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker network create tedge ||:
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60

    # Create data directory
    DeviceLibrary.Execute Command    mkdir /data

Create Container
    [Arguments]    ${name}    ${image}
    ${operation}=    Cumulocity.Install Software    {"name": "${name}", "version": "${image}", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "${name}", "version": "${image}", "softwareType": "container"}
