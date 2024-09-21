*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary

Suite Setup    Set Main Device

*** Test Cases ***

Service status
    Cumulocity.Should Have Services    name=tedge-container-monitor      service_type=service    status=up    timeout=90

    # Restarting mosquitto should not cause the service to be shown as "down"
    # https://github.com/thin-edge/tedge-container-plugin/issues/37
    Cumulocity.Execute Shell Command    sudo systemctl stop mosquitto
    Sleep    2s    reason=Wait before starting mosquitto
    Cumulocity.Execute Shell Command    sudo systemctl start mosquitto
    Sleep    5s    reason=Give time for server to process any status changes to prevent checking too early
    Cumulocity.Should Have Services    name=tedge-container-monitor      service_type=service    status=up

Sends measurements
    Skip    TODO
    ${date_from}=    Get Test Start Time
    Cumulocity.Device Should Have Measurements    minimum=1    maximum=1    type=environment    after=${date_from}
