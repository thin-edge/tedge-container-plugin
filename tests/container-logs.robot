*** Settings ***
Resource    ./resources/common.robot
Library    String
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Suite Setup    Suite Setup

Test Tags    docker    podman

*** Test Cases ***

Get Container Logs
    # Run dummy container then exit
    DeviceLibrary.Execute Command    cmd=tedge-container engine docker run --name app10 httpd:2.4.61-alpine sh -c 'echo hello inside container stdout; echo hello inside container stderr >&2;'

    # Fetch logs
    ${output}=    DeviceLibrary.Execute Command    sudo tedge-container tools container-logs app10
    Should Contain    ${output}    hello inside container stdout
    Should Contain    ${output}    hello inside container stderr

Get Container Logs with only last N lines
    # Run dummy container then exit
    DeviceLibrary.Execute Command    cmd=tedge-container engine docker run --name app11 httpd:2.4.61-alpine sh -c 'echo hello inside container stdout; echo hello inside container stderr >&2;'

    # Fetch logs
    ${output}=    DeviceLibrary.Execute Command    sudo tedge-container tools container-logs app11 -n 1

    ${total_lines}=    String.Get Line Count    ${output}
    Should Be Equal As Integers    ${total_lines}    1

    Should Not Contain    ${output}    hello inside container stdout
    Should Contain    ${output}    hello inside container stderr

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
