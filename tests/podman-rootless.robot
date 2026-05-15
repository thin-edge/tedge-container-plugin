*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Suite Setup    Suite Setup
Test Teardown    Collect Logs
Test Tags    podman    docker

*** Test Cases ***

Clone a rootless container
    Execute Command    cmd=rm -f /tmp/info && mkfifo /tmp/info && chown tedge:tedge /tmp/info && chmod 660 /tmp/info && mkdir -p /home/tedge && chown tedge:tedge -R /home/tedge
    Execute Command    cmd=sudo usermod --add-subuids 100000-165535 tedge && sudo usermod --add-subgids 100000-165535 tedge && sudo usermod -s /bin/bash tedge
    Execute Command    cmd=sudo -iu tedge systemctl --user start podman.service && loginctl enable-linger tedge
    Execute Command    cmd=sudo -iu tedge podman run --pids-limit=-1 -d -t -u 1000 --name test01 -v /tmp/info:/tmp/info --userns keep-id alpine sh -c 'while true; do echo hello > /tmp/info; sleep 1; done'
    Execute Command    cmd=sudo -iu tedge tedge-container tools container-clone --container test01    retries=0

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
