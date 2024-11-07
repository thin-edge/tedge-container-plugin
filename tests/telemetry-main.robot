*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Test Setup    Test Setup

*** Test Cases ***

Service status
    Cumulocity.Should Have Services    name=tedge-container-plugin      service_type=service    status=up    timeout=90

    # Restarting mosquitto should not cause the service to be shown as "down"
    # https://github.com/thin-edge/tedge-container-plugin/issues/37
    Cumulocity.Execute Shell Command    sudo systemctl stop mosquitto
    Sleep    2s    reason=Wait before starting mosquitto
    Cumulocity.Execute Shell Command    sudo systemctl start mosquitto
    Sleep    5s    reason=Give time for server to process any status changes to prevent checking too early
    Cumulocity.Should Have Services    name=tedge-container-plugin      service_type=service    status=up

Sends measurements
    ${date_from}=    Get Test Start Time
    Install Example Container
    ${SERVICE_SN}=    Get Service External ID    ${DEVICE_SN}    customapp1
    Cumulocity.External Identity Should Exist    ${SERVICE_SN}
    ${measurements}=    Cumulocity.Device Should Have Measurements    minimum=1    type=resource_usage    after=${date_from}    timeout=120

    [Teardown]    Uninstall Example Container

*** Keywords ***

Test Setup
    ${DEVICE_SN}=    Setup
    Set Suite Variable    $DEVICE_SN
    Cumulocity.External Identity Should Exist    ${DEVICE_SN}

Install Example Container
    ${operation}=    Cumulocity.Install Software    {"name": "customapp1", "version": "httpd:2.4", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60

Uninstall Example Container
    Cumulocity.Set Managed Object    ${DEVICE_SN}
    ${operation}=    Cumulocity.Uninstall Software    {"name": "customapp1", "version": "httpd:2.4", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60

Get Service External ID
    [Arguments]    ${device_sn}    ${service_name}
    RETURN    ${device_sn}:device:main:service:${service_name}
