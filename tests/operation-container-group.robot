*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Suite Setup    Suite Setup
Test Teardown    Collect Logs

*** Test Cases ***

Install/uninstall container-group package
    [Tags]    podman    docker
    ${binary_url}=    Cumulocity.Create Inventory Binary    nginx    container-group    file=${CURDIR}/data/docker-compose.nginx.yaml
    ${operation}=    Cumulocity.Install Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group"}
    ${operation}=    Cumulocity.Execute Shell Command    wget -O- 127.0.0.1:8080
    Operation Should Be SUCCESSFUL    ${operation}
    Should Contain    ${operation.to_json()["c8y_Command"]["result"]}    Welcome to nginx
    Cumulocity.Should Have Services    name=nginx@nginx    service_type=container-group    status=up

    # Check if you can request the logs for it
    Cumulocity.Should Contain Supported Log Types    nginx@nginx::container-group
    ${operation}=    Cumulocity.Get Log File    nginx@nginx::container-group
    Operation Should Be SUCCESSFUL    ${operation}

    # Uninstall
    ${operation}=     Cumulocity.Uninstall Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Not Have Installed Software    nginx
    Cumulocity.Should Have Services    name=nginx@nginx    service_type=container-group    min_count=0    max_count=0

Install/uninstall container-group package with custom project name and without using module name in services
    [Tags]    docker    docker-modern
    # Note: older versions of podman-compose don't support the .name property in the compose, it was added around podman-compose >= 1.3.0
    Install/uninstall container-group package with custom project name    module_name=app7-dev    service_name=app7@nginx    use_module_name=false

Install/uninstall container-group package with custom project name and with using module name in services
    [Tags]    docker    docker-modern
    # Note: older versions of podman-compose don't support the .name property in the compose, it was added around podman-compose >= 1.3.0
    Install/uninstall container-group package with custom project name    module_name=app7-test    service_name=app7-test@nginx    use_module_name=true

Install/uninstall container-group package with non-existent image
    [Tags]    podman    docker
    ${binary_url}=    Cumulocity.Create Inventory Binary    nginx    container-group    file=${CURDIR}/data/docker-compose.invalid-image.yaml
    ${operation}=    Cumulocity.Install Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be FAILED    ${operation}    timeout=120

Install invalid container-group
    [Tags]    podman    docker
    ${binary_url}=    Cumulocity.Create Inventory Binary    nginx    container-group    file=${CURDIR}/data/docker-compose.invalid.yaml
    ${operation}=    Cumulocity.Install Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be FAILED    ${operation}    timeout=120
    Device Should Not Have Installed Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group"}

Install container-group with multiple files - app1
    [Tags]    podman    docker
    Install container-group file    app1    1.0.1    app1    ${CURDIR}/data/apps/app1.tar.gz

Install container-group with multiple files - app2
    [Tags]    podman    docker
    Install container-group file    app2    1.2.3    app2    ${CURDIR}/data/apps/app2.zip

Install container group that uses host volume mount
    [Tags]    podman    docker
    [Setup]    Start Service    tedge-container-plugin
    # Install container-group
    Install container-group application    app5    1.0.0    app5    ${CURDIR}/data/apps/app5.tar.gz
    Device Should Have Installed Software    {"name": "app5", "version": "1.0.0", "softwareType": "container-group"}
    Cumulocity.Should Have Services    name=app5@httpd    service_type=container-group    status=up

    ${operation}=    Cumulocity.Execute Shell Command    text=curl -sf http://127.0.0.1:9082
    ${operation}=    Operation Should Be SUCCESSFUL    ${operation}
    Should Contain    ${operation["c8y_Command"]["result"]}    It works

Install container group with a container in a crash loop
    # Retries are required as the container can sometimes fail during installation instead of after
    [Tags]    podman    docker    test:retry(3)
    [Setup]    Start Service    tedge-container-plugin

    ${binary_url}=    Cumulocity.Create Inventory Binary    crash-loop    container-group    file=${CURDIR}/data/docker-compose.crash-loop.yaml
    ${operation}=    Cumulocity.Install Software    {"name": "crash-loop", "version": "1.0.0", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60

    # Install container-group
    Install container-group application    crash-loop    1.0.0    crash-loop    ${CURDIR}/data/docker-compose.crash-loop.yaml
    Device Should Have Installed Software    {"name": "crash-loop", "version": "1.0.0", "softwareType": "container-group"}
    Cumulocity.Should Have Services    name=crash-loop@app    service_type=container-group    status=down

    Cumulocity.Set Managed Object    external_id=${DEVICE_SN}:device:main:service:crash-loop@app
    Cumulocity.Device Should Have Alarm/s    type=ContainerCrashLoop    expected_text=Container is in a crash loop

    # Uninstall
    Cumulocity.Set Managed Object    external_id=${DEVICE_SN}
    ${operation}=     Cumulocity.Uninstall Software    {"name": "crash-loop", "version": "1.0.0", "softwareType": "container-group"}
    Cumulocity.Operation Should Be SUCCESSFUL    ${operation}
    Cumulocity.Device Should Not Have Installed Software    crash-loop
    Cumulocity.Should Have Services    name=crash-loop@app    service_type=container-group    min_count=0    max_count=0

Podman gateway host is added by default
    [Tags]    podman    docker
    Install container-group and access gateway host    app8    service=app8@mqtt-client    file=${CURDIR}/data/docker-compose.app8-mqtt-client.yaml

Docker gateway host is added by default
    [Tags]    podman    docker
    Install container-group and access gateway host    app9    service=app9@mqtt-client    file=${CURDIR}/data/docker-compose.app9-docker-host.yaml

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

Install container-group file
    [Arguments]    ${package_name}    ${package_version}    ${service_name}    ${file}
    ${binary_url}=    Cumulocity.Create Inventory Binary    ${package_name}    container-group    file=${file}
    ${operation}=    Cumulocity.Install Software    {"name": "${package_name}", "version": "${package_version}", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=300

    DeviceLibrary.Directory Should Not Be Empty    /data/tedge-container-plugin/compose/${package_name}

    Device Should Have Installed Software    {"name": "${package_name}", "version": "${package_version}", "softwareType": "container-group"}
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker run --rm -t --network tedge docker.io/library/busybox wget -O- ${service_name}:80
    Operation Should Be SUCCESSFUL    ${operation}
    Should Contain    ${operation.to_json()["c8y_Command"]["result"]}    My Custom Web Application
    Cumulocity.Should Have Services    name=${package_name}@${service_name}    service_type=container-group    status=up

Install container-group application
    [Documentation]    Install a container-group and let the user do follow up tests
    [Arguments]    ${package_name}    ${package_version}    ${service_name}    ${file}
    ${binary_url}=    Cumulocity.Create Inventory Binary    ${package_name}    container-group    file=${file}
    ${operation}=    Cumulocity.Install Software    {"name": "${package_name}", "version": "${package_version}", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=300

Install/uninstall container-group package with custom project name
    [Arguments]    ${module_name}    ${service_name}    ${use_module_name}
    DeviceLibrary.Execute Command    cmd=sed -i '/^\[container_group\]/,/^\[.*\]/ { s/^use_module_name = .*$/use_module_name = ${use_module_name}/ }' /etc/tedge/plugins/tedge-container-plugin.toml
    DeviceLibrary.Restart Service    tedge-container-plugin

    ${binary_url}=    Cumulocity.Create Inventory Binary    ${module_name}    container-group    file=${CURDIR}/data/docker-compose.app7-dev.yaml
    ${operation}=    Cumulocity.Install Software    {"name": "${module_name}", "version": "1.0.0", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "${module_name}", "version": "1.0.0", "softwareType": "container-group"}
    Cumulocity.Should Have Services    name=${service_name}    service_type=container-group    status=up

    # Check if you can request the logs for it
    Cumulocity.Should Contain Supported Log Types    ${service_name}::container-group
    ${operation}=    Cumulocity.Get Log File    ${service_name}::container-group
    Operation Should Be SUCCESSFUL    ${operation}

    # Uninstall
    ${operation}=     Cumulocity.Uninstall Software    {"name": "${module_name}", "version": "1.0.0", "softwareType": "container-group"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Not Have Installed Software    ${module_name}
    Cumulocity.Should Have Services    name=${service_name}    service_type=container-group    min_count=0    max_count=0

Install container-group and access gateway host
    [Arguments]    ${name}    ${service}    ${file}
    ${binary_url}=    Cumulocity.Create Inventory Binary    ${name}    container-group    file=${file}
    ${operation}=    Cumulocity.Install Software    {"name": "${name}", "version": "1.0.0", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60

    Sleep    5s    reason=Give time for the service to fail
    Cumulocity.Should Have Services    name=${service}    service_type=container-group    status=up
