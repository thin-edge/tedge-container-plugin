*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Test Setup    Test Setup
Test Teardown    Collect Logs

*** Test Cases ***

Update to tedge-container-plugin-ng
    DeviceLibrary.Execute Command   cmd=apt-get install -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confnew" /opt/packages/*.deb
    Cumulocity.Should Have Services    name=tedge-container-plugin    service_type=service    min_count=1    max_count=1    timeout=30
    Cumulocity.Should Have Services    name=tedge-container-monitor    service_type=service    min_count=0    max_count=0    timeout=30

    DeviceLibrary.Execute Command    cmd=systemctl status tedge-container-monitor   exp_exit_code=!0

    # Remove package
    DeviceLibrary.Execute Command   cmd=apt-get remove -y tedge-container-plugin-ng
    DeviceLibrary.Execute Command    cmd=systemctl status tedge-container-plugin   exp_exit_code=!0

*** Keywords ***

Test Setup
    ${DEVICE_SN}=    Setup
    Set Suite Variable    $DEVICE_SN
    Cumulocity.External Identity Should Exist    ${DEVICE_SN}

    # Install older version
    DeviceLibrary.Execute Command    apt-get remove -y tedge-container-plugin-ng && apt-get purge -y tedge-container-plugin-ng
    DeviceLibrary.Execute Command    apt-get remove -y tedge-container-plugin && apt-get purge -y tedge-container-plugin     ignore_exit_code=${True}

    DeviceLibrary.Execute Command   apt-get update && apt-get install -y tedge-container-plugin
    Cumulocity.Should Have Services    name=tedge-container-monitor    service_type=service    min_count=1    max_count=1    timeout=60

Collect Logs
    Collect Workflow Logs
    Collect Systemd Logs

Collect Systemd Logs
    Execute Command    sudo journalctl -n 10000

Collect Workflow Logs
    Execute Command    cat /var/log/tedge/agent/*
