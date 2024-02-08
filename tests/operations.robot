*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary

Suite Setup    Suite Setup

*** Test Cases ***

Get Configuration
    ${binary_url}    Cumulocity.Create Inventory Binary    name    binary_type    file=${CURDIR}/data/tedge-configuration-plugin.toml
    ${operation}=    Cumulocity.Set Configuration    tedge-configuration-plugin    ${binary_url}
    Operation Should Be SUCCESSFUL    ${operation}

    ${operation}=    Cumulocity.Get Configuration    typename=tedge-container-plugin
    Operation Should Be SUCCESSFUL    ${operation}

Install container-group package
    ${binary_url}=    Cumulocity.Create Inventory Binary    nginx    container-group    file=${CURDIR}/data/docker-compose.nginx.yaml
    ${operation}=    Cumulocity.Install Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Have Installed Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group"}
    ${operation}=    Cumulocity.Execute Shell Command    wget -O- nginx:80
    Operation Should Be SUCCESSFUL    ${operation}
    Should Contain    ${operation.to_json()["c8y_Command"]["result"]}    Welcome to nginx
    Cumulocity.Should Have Services    name=nginx@nginx    service_type=container-group    status=up

Install container-group with multiple files
    [Template]    Install container-group file
    app1    1.0.1    app1    ${CURDIR}/data/apps/app1.tar.gz
    app2    1.2.3    app2    ${CURDIR}/data/apps/app2.zip

Uninstall container-group
    ${operation}=     Cumulocity.Uninstall Software    {"name": "nginx", "version": "1.0.0", "softwareType": "container-group"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Not Have Installed Software    nginx
    Cumulocity.Should Have Services    name=nginx@nginx    service_type=container-group    status=uninstalled

Install container package
    ${operation}=    Cumulocity.Install Software    {"name": "webserver", "version": "httpd:2.4", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Have Installed Software    {"name": "webserver", "version": "httpd:2.4", "softwareType": "container"}
    ${operation}=    Cumulocity.Execute Shell Command    wget -O- webserver:80
    Operation Should Be SUCCESSFUL    ${operation}
    Should Contain    ${operation.to_json()["c8y_Command"]["result"]}    It works!

Uninstall container package
    ${operation}=     Cumulocity.Uninstall Software    {"name": "webserver", "version": "httpd:2.4", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Not Have Installed Software    webserver


*** Keywords ***

Suite Setup
    Set Main Device

Install container-group file
    [Arguments]    ${package_name}    ${package_version}    ${service_name}    ${file}
    ${binary_url}=    Cumulocity.Create Inventory Binary    ${package_name}    container-group    file=${file}
    ${operation}=    Cumulocity.Install Software    {"name": "${package_name}", "version": "${package_version}", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Have Installed Software    {"name": "${package_name}", "version": "${package_version}", "softwareType": "container-group"}
    ${operation}=    Cumulocity.Execute Shell Command    wget -O- ${service_name}:80
    Operation Should Be SUCCESSFUL    ${operation}
    Should Contain    ${operation.to_json()["c8y_Command"]["result"]}    My Custom Web Application
    Cumulocity.Should Have Services    name=${package_name}@${service_name}    service_type=container-group    status=up
