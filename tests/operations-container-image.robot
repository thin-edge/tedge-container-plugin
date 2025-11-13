*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Suite Setup    Suite Setup
Test Teardown    Collect Logs
Test Tags    podman    docker

*** Test Cases ***

Install/uninstall container-image
    ${operation}=    Cumulocity.Install Software    {"name": "docker.io/library/httpd", "version": "2.4.64", "softwareType": "container-image", "url": ""}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "docker.io/library/httpd", "version": "2.4.64", "softwareType": "container-image"}

    # install another version of the same image
    ${operation}=    Cumulocity.Install Software    {"name": "docker.io/library/httpd", "version": "2.4.65", "softwareType": "container-image", "url": ""}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "docker.io/library/httpd", "version": "2.4.65", "softwareType": "container-image"}

    # Uninstall
    ${operation}=     Cumulocity.Uninstall Software    {"name": "docker.io/library/httpd", "version": "2.4.64", "softwareType": "container-image"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Not Have Installed Software    docker.io/library/httpd,2.4.64

    # The other version should be left untouched
    Device Should Have Installed Software    {"name": "docker.io/library/httpd", "version": "2.4.65", "softwareType": "container-image"}

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
