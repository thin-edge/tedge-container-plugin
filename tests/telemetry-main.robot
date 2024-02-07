*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary

Suite Setup    Set Main Device

*** Test Cases ***

Service status
    Cumulocity.Should Have Services    name=tedge-container-monitor      service_type=service    status=up    timeout=90

Sends measurements
    Skip    TODO
    ${date_from}=    Get Test Start Time
    Cumulocity.Device Should Have Measurements    minimum=1    maximum=1    type=environment    after=${date_from}
