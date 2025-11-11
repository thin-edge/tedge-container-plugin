*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Suite Setup    Suite Setup
Test Teardown    Collect Logs
Test Tags    podman    docker

*** Test Cases ***

Clone a rootless container
    Execute Command    rm -f /tmp/info && mkfifo /tmp/info && chown tedge:tedge /tmp/info && chmod 660 /tmp/info
    Execute Command    sudo -u tedge podman rm -f test01
    Execute Command    sudo -u tedge podman run -d -it -u tedge --name test01 -v /tmp/info:/tmp/info --userns keep-id alpine sh -c 'while true; do echo hello > /tmp/info; sleep 1; done'
    Execute Command    sudo -u tedge tedge-container tools container-clone --container test01
    Log    done


*** Keywords ***

Suite Setup
    ${DEVICE_SN}=    Setup
    Set Suite Variable    $DEVICE_SN
    Cumulocity.External Identity Should Exist    ${DEVICE_SN}
    Cumulocity.Should Have Services    name=tedge-container-plugin    service_type=service    min_count=1    max_count=1    timeout=30

    # Create common network for all containers
    ${operation}=    Cumulocity.Execute Shell Command    set -a; . /etc/tedge-container-plugin/env; docker network create tedge ||:

    # Create data directory
    DeviceLibrary.Execute Command    mkdir /data
