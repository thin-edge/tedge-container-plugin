*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Test Setup    Test Setup
Test Teardown    Collect Logs

*** Variables ***

${PRIVATE_IMAGE}            %{PRIVATE_IMAGE=}
${REGISTRY1_REPO}           %{REGISTRY1_REPO=docker.io}
${REGISTRY1_USERNAME}       %{REGISTRY1_USERNAME=}
${REGISTRY1_PASSWORD}       %{REGISTRY1_PASSWORD=}

*** Test Cases ***

Install/uninstall container package from private repository - credentials file
    [Tags]    docker    podman
    DeviceLibrary.Execute Command    
    ...    cmd=printf -- '[registry1]\nrepo = "${REGISTRY1_REPO}"\nusername = "${REGISTRY1_USERNAME}"\npassword = "${REGISTRY1_PASSWORD}"\n' > /data/tedge-container-plugin/credentials.toml

    ${operation}=    Cumulocity.Install Software    {"name": "testapp1", "version": "${PRIVATE_IMAGE}", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "testapp1", "version": "${PRIVATE_IMAGE}", "softwareType": "container"}

Install/uninstall container package from private repository - credentials script
    [Tags]    docker    podman
    Transfer To Device    ${CURDIR}/data/registry-credentials    /usr/bin/
    DeviceLibrary.Execute Command    cmd=sed -i 's|@@USERNAME@@|${REGISTRY1_USERNAME}|g' /usr/bin/registry-credentials
    DeviceLibrary.Execute Command    cmd=sed -i 's|@@PASSWORD@@|${REGISTRY1_PASSWORD}|g' /usr/bin/registry-credentials

    ${operation}=    Cumulocity.Install Software    {"name": "testapp2", "version": "${PRIVATE_IMAGE}", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "testapp2", "version": "${PRIVATE_IMAGE}", "softwareType": "container"}

Install/uninstall container package from private repository - credentials script with cache
    [Tags]    docker    podman
    Transfer To Device    ${CURDIR}/data/registry-credentials-with-cache    /usr/bin/registry-credentials
    DeviceLibrary.Execute Command    cmd=sed -i 's|@@USERNAME@@|${REGISTRY1_USERNAME}|g' /usr/bin/registry-credentials
    DeviceLibrary.Execute Command    cmd=sed -i 's|@@PASSWORD@@|${REGISTRY1_PASSWORD}|g' /usr/bin/registry-credentials

    ${operation}=    Cumulocity.Install Software    {"name": "testapp2", "version": "${PRIVATE_IMAGE}", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "testapp2", "version": "${PRIVATE_IMAGE}", "softwareType": "container"}

Install/uninstall container package from private repository - engine credentials
    [Documentation]    login to registry from host
    [Tags]    podman
    DeviceLibrary.Execute Command    tedge-container engine docker login ${REGISTRY1_REPO} -u '${REGISTRY1_USERNAME}' --password '${REGISTRY1_PASSWORD}'
    ${operation}=    Cumulocity.Install Software    {"name": "testapp3", "version": "${PRIVATE_IMAGE}", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "testapp3", "version": "${PRIVATE_IMAGE}", "softwareType": "container"}

Install/uninstall container package from private repository - docker from docker
    [Documentation]    login inside a container with the auth file mounted from the host
    [Tags]    podman

    # Start a container
    DeviceLibrary.Execute Command    mkdir -p /run/containers/0/
    Install container-group file    app4    2.0.0    app4    ${CURDIR}/data/apps/app4.tar.gz
    Device Should Have Installed Software    {"name": "app4", "version": "2.0.0", "softwareType": "container-group"}
    Cumulocity.Should Have Services    name=app4@main    service_type=container-group    status=up

    # Deploy a new container from inside the container
    DeviceLibrary.Execute Command    tedge-container engine docker exec app4_main_1 sudo tedge-container engine docker login ${REGISTRY1_REPO} -u '${REGISTRY1_USERNAME}' --password '${REGISTRY1_PASSWORD}' --authfile /run/containers/0/auth.json
    DeviceLibrary.Execute Command    tedge-container engine docker exec app4_main_1 sudo tedge-container container install app5 --module-version ${PRIVATE_IMAGE}

*** Keywords ***

Test Setup
    ${DEVICE_SN}=    Setup
    Set Suite Variable    $DEVICE_SN
    Cumulocity.External Identity Should Exist    ${DEVICE_SN}
    Cumulocity.Should Have Services    name=tedge-container-plugin    service_type=service    min_count=1    max_count=1    timeout=30

    # Create common network for all containers
    ${operation}=    Cumulocity.Execute Shell Command    set -a; . /etc/tedge-container-plugin/env; docker network create tedge ||:

    # Create data directory
    DeviceLibrary.Execute Command    mkdir -p /data/tedge-container-plugin/


Install container-group file
    [Arguments]    ${package_name}    ${package_version}    ${service_name}    ${file}
    ${binary_url}=    Cumulocity.Create Inventory Binary    ${package_name}    container-group    file=${file}
    ${operation}=    Cumulocity.Install Software    {"name": "${package_name}", "version": "${package_version}", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=300


Collect Logs
    Collect Workflow Logs
    Collect Systemd Logs

Collect Systemd Logs
    Execute Command    sudo journalctl -n 10000

Collect Workflow Logs
    Execute Command    cat /var/log/tedge/agent/*
