*** Settings ***
Resource    ./resources/common.robot
Library    String
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Suite Setup    Suite Setup

Test Tags    docker    podman

*** Test Cases ***

Restart Container
    ${started_at_before}=    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker container inspect app40 --format "{{ .State.StartedAt }}"
    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools container-restart app40
    ${started_at_after}=    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker container inspect app40 --format "{{ .State.StartedAt }}"
    Should Not Be Equal As Strings    ${started_at_after}    ${started_at_before}

Restart Unknown Container Fails
    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools container-restart doesNotExist    exp_exit_code=!0

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

    # Create a dummy container
    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools container-remove app40 ||: ; sudo tedge-container engine docker run -d --network bridge --name app40 httpd:2.4.61-alpine
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker container inspect app40    exp_exit_code=0
