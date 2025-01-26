*** Settings ***
Resource    ./resources/common.robot
Library    String
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Suite Setup    Suite Setup

Test Tags    docker    podman

*** Test Cases ***

Remove Container
    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools container-remove app30 ||: ; sudo tedge-container engine docker run -d --network bridge --name app30 httpd:2.4.61-alpine
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker container inspect app30    exp_exit_code=0

    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools container-remove app30
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker container inspect app30    exp_exit_code=!0

Remove Container Non Existent Container Should Not Through An Error
    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools container-remove app31

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
