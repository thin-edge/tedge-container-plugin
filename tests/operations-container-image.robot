*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Suite Setup    Suite Setup
Test Teardown    Collect Logs
Test Tags    podman    docker

*** Test Cases ***

Install/uninstall container-image
    [Documentation]    Older docker versions < v28, the container image will only be published as httpd,
    ...    and not docker.io/library/httpd so the conversion is lossy, therefore dynamically set the image
    ...    name to be used in the test based on the installed docker version.
    ${IMAGE_NAME}=    Execute Command    cmd=if which podman >/dev/null 2>&1; then echo "docker.io/library/httpd"; else (docker --version 2>/dev/null | grep -q "version 2[0-7]" && echo "httpd" || echo "docker.io/library/httpd"); fi    strip=${True}
    ${operation}=    Cumulocity.Install Software    {"name": "${IMAGE_NAME}", "version": "2.4.64", "softwareType": "container-image", "url": ""}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "${IMAGE_NAME}", "version": "2.4.64", "softwareType": "container-image"}

    # install another version of the same image
    ${operation}=    Cumulocity.Install Software    {"name": "${IMAGE_NAME}", "version": "2.4.65", "softwareType": "container-image", "url": ""}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "${IMAGE_NAME}", "version": "2.4.65", "softwareType": "container-image"}

    # Uninstall
    ${operation}=     Cumulocity.Uninstall Software    {"name": "${IMAGE_NAME}", "version": "2.4.64", "softwareType": "container-image"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Not Have Installed Software    ${IMAGE_NAME},2.4.64

    # The other version should be left untouched
    Device Should Have Installed Software    {"name": "${IMAGE_NAME}", "version": "2.4.65", "softwareType": "container-image"}

Install/uninstall not existent container image
    ${operation}=    Cumulocity.Install Software    {"name": "doesnotexist", "version": "0.0.0-1201872", "softwareType": "container-image", "url": ""}
    Operation Should Be FAILED    ${operation}    timeout=120

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

    # enable sm-plugin
    # DeviceLibrary.Execute Command    cmd=echo CONTAINER_CONTAINER_IMAGE_ENABLED=true >> /etc/tedge-container-plugin/env
    # enable the sm-plugin by default
    DeviceLibrary.Execute Command    cmd=sed -i '/^\[container_image\]/,/^\[.*\]/ { s/^enabled = .*$/enabled = true/ }' /etc/tedge/plugins/tedge-container-plugin.toml
    DeviceLibrary.Restart Service    tedge-agent
