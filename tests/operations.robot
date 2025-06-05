*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Suite Setup    Suite Setup
Test Teardown    Collect Logs
Test Tags    podman    docker

*** Test Cases ***

Get Configuration
    ${binary_url}    Cumulocity.Create Inventory Binary    name    binary_type    file=${CURDIR}/data/tedge-configuration-plugin.toml
    ${operation}=    Cumulocity.Set Configuration    tedge-configuration-plugin    ${binary_url}
    Operation Should Be SUCCESSFUL    ${operation}

    ${operation}=    Cumulocity.Get Configuration    typename=tedge-container-plugin
    Operation Should Be SUCCESSFUL    ${operation}

Install/uninstall container-group package
    ${binary_url}=    Cumulocity.Create Inventory Binary    nginx    container-group    file=${CURDIR}/data/docker-compose.nginx.yaml
    ${operation}=    Cumulocity.Install Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group"}
    ${operation}=    Cumulocity.Execute Shell Command    wget -O- 127.0.0.1:8080
    Operation Should Be SUCCESSFUL    ${operation}
    Should Contain    ${operation.to_json()["c8y_Command"]["result"]}    Welcome to nginx
    Cumulocity.Should Have Services    name=nginx@nginx    service_type=container-group    status=up

    # Uninstall
    ${operation}=     Cumulocity.Uninstall Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Not Have Installed Software    nginx
    Cumulocity.Should Have Services    name=nginx@nginx    service_type=container-group    min_count=0    max_count=0

Install/uninstall container-group package with non-existent image
    ${binary_url}=    Cumulocity.Create Inventory Binary    nginx    container-group    file=${CURDIR}/data/docker-compose.invalid-image.yaml
    ${operation}=    Cumulocity.Install Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be FAILED    ${operation}    timeout=60

Install invalid container-group
    ${binary_url}=    Cumulocity.Create Inventory Binary    nginx    container-group    file=${CURDIR}/data/docker-compose.invalid.yaml
    ${operation}=    Cumulocity.Install Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be FAILED    ${operation}    timeout=60
    Device Should Not Have Installed Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group"}

Install container-group with multiple files - app1
    Install container-group file    app1    1.0.1    app1    ${CURDIR}/data/apps/app1.tar.gz

Install container-group with multiple files - app2
    Install container-group file    app2    1.2.3    app2    ${CURDIR}/data/apps/app2.zip

Install/uninstall container package
    ${operation}=    Cumulocity.Install Software    {"name": "webserver", "version": "docker.io/library/httpd:2.4", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "webserver", "version": "docker.io/library/httpd:2.4", "softwareType": "container"}
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker run --rm -t --network tedge docker.io/library/busybox wget -O- webserver:80;
    Operation Should Be SUCCESSFUL    ${operation}
    Should Contain    ${operation.to_json()["c8y_Command"]["result"]}    It works!
    Cumulocity.Should Have Services    name=webserver    service_type=container    status=up

    # Uninstall
    ${operation}=     Cumulocity.Uninstall Software    {"name": "webserver", "version": "docker.io/library/httpd:2.4", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Not Have Installed Software    webserver
    Cumulocity.Should Have Services    name=webserver    service_type=container    min_count=0    max_count=0


Install/uninstall container package from file
    ${binary_url}=    Cumulocity.Create Inventory Binary    app3    container    file=${CURDIR}/data/apps/app3.tar

    ${operation}=    Cumulocity.Install Software    {"name": "app3", "version": "docker.io/library/app3:latest", "softwareType": "container", "url": "${binary_url}"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Have Installed Software    {"name": "app3", "version": "docker.io/library/app3:latest", "softwareType": "container"}
    Cumulocity.Should Have Services    name=app3    service_type=container    status=up

    # Uninstall
    ${operation}=     Cumulocity.Uninstall Software    {"name": "app3", "version": "docker.io/library/app3:latest", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Not Have Installed Software    app3
    Cumulocity.Should Have Services    name=app3    service_type=container    min_count=0    max_count=0


Manual container creation/deletion
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker network create tedge ||:; sudo tedge-container engine docker run -d --network tedge --name manualapp1 httpd:2.4
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60

    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker run --rm -t --network tedge docker.io/library/busybox wget -O- manualapp1:80;
    Operation Should Be SUCCESSFUL    ${operation}

    Should Contain    ${operation.to_json()["c8y_Command"]["result"]}    It works!
    Cumulocity.Should Have Services    name=manualapp1    service_type=container    status=up

    # Pause
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker pause manualapp1;
    Operation Should Be SUCCESSFUL    ${operation}
    Cumulocity.Should Have Services    name=manualapp1    service_type=container    status=down

    # Unpause
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker unpause manualapp1;
    Operation Should Be SUCCESSFUL    ${operation}
    Cumulocity.Should Have Services    name=manualapp1    service_type=container    status=up

    # Uninstall
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker rm manualapp1 --force
    Operation Should Be SUCCESSFUL    ${operation}
    Cumulocity.Should Have Services    name=manualapp1    service_type=container    min_count=0    max_count=0    timeout=10


Manual container creation/deletion with error on run
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker run -d --name manualapp2 httpd:2.4 --invalid-arg || exit 0
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Cumulocity.Should Have Services    name=manualapp2    service_type=container    status=down

    # Uninstall
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker rm manualapp2 --force
    Operation Should Be SUCCESSFUL    ${operation}
    Cumulocity.Should Have Services    name=manualapp2    service_type=container    min_count=0    max_count=0    timeout=10


Manual container created and then killed
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker run -d --name manualapp3 busybox sh -c 'exec sleep infinity'
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Cumulocity.Should Have Services    name=manualapp3    service_type=container    status=up

    # Manually kill the container's PID 1
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker kill -s KILL manualapp3
    Cumulocity.Should Have Services    name=manualapp3    service_type=container    status=down

    # Uninstall
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker rm manualapp3 --force
    Operation Should Be SUCCESSFUL    ${operation}
    Cumulocity.Should Have Services    name=manualapp3    service_type=container    min_count=0    max_count=0    timeout=10


Remove Orphaned Cloud Services
    [Documentation]    Orphaned cloud services can occur if entities are deregistered manually when the tedge-container-plugin
    ...    service is not running.
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker run -d --name manualapp4 busybox sh -c 'exec sleep infinity'
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Cumulocity.Should Have Services    name=manualapp4    service_type=container    status=up

    Stop Service    tedge-container-plugin

    # Uninstall
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker rm manualapp4 --force
    Operation Should Be SUCCESSFUL    ${operation}

    # Clear container service locally
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge mqtt pub -r 'te/device/main/service/manualapp4' ''; sleep 1; sudo tedge http delete '/te/v1/entities/device/main/service/manualapp4'
    Operation Should Be SUCCESSFUL    ${operation}

    # Confirm that the cloud's service status has not changed
    # Note: This could change once thin-edge.io supports deleting the cloud entities
    Sleep    1s
    Cumulocity.Should Have Services    name=manualapp4    service_type=container    status=up

    # Start the service, and check that the service has been removed (without the explicit service type defined)
    Start Service    tedge-container-plugin
    Cumulocity.Should Have Services    name=manualapp4    min_count=0    max_count=0    timeout=10

Install container group that uses host volume mount
    # Install container-group
    Install container-group application    app5    1.0.0    app5    ${CURDIR}/data/apps/app5.tar.gz
    Device Should Have Installed Software    {"name": "app5", "version": "1.0.0", "softwareType": "container-group"}
    Cumulocity.Should Have Services    name=app5@httpd    service_type=container-group    status=up

    ${operation}=    Cumulocity.Execute Shell Command    text=curl -sf http://127.0.0.1:9082
    ${operation}=    Operation Should Be SUCCESSFUL    ${operation}
    Should Contain    ${operation["c8y_Command"]["result"]}    It works

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
