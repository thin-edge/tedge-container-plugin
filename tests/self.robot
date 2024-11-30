*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Suite Setup    Suite Setup
Test Teardown    Collect Logs

Test Tags    docker    podman

*** Test Cases ***

Self Update Is Present Using Self Type
    ${output}=    DeviceLibrary.Execute Command    cmd=tedge-container self check '[{"type":"self","modules":[{"name":"tedge","action":"install","version":"123"}]},{"type":"container","modules":[{"name":"foo","version":"bar:1.0.0","action":"install"}]}]' --container tedge    strip=${True}
    Should Contain    ${output}    {"containerName":"tedge","image":"123","updateList":[{"type":"container","modules":[{"name":"foo","version":"bar:1.0.0","action":"install"}]}]}

Self Update Is Present Using Container Type
    ${output}=    DeviceLibrary.Execute Command    cmd=tedge-container self check '[{"type":"container","modules":[{"name":"tedge","action":"install","version":"123"},{"name":"foo","action":"install","version":"bar:latest","url":"https://foobar.com/example"}]}]' --container tedge    strip=${True}
    Should Contain    ${output}    {"containerName":"tedge","image":"123","updateList":[{"type":"container","modules":[{"name":"foo","version":"bar:latest","url":"https://foobar.com/example","action":"install"}]}]}

Self Update Is Not Present
    ${output}=    DeviceLibrary.Execute Command    cmd=tedge-container self check '[{"type":"custom","modules":[{"name":"foo","action":"install","version":"bar:latest","url":"https://foobar.com/example"}]}]' --container tedge    exp_exit_code=1


*** Keywords ***

Suite Setup
    ${DEVICE_SN}=    Setup
    Set Suite Variable    $DEVICE_SN
    Cumulocity.External Identity Should Exist    ${DEVICE_SN}
    Cumulocity.Should Have Services    name=tedge-container-plugin    service_type=service    min_count=1    max_count=1    timeout=30

    # Create data directory
    DeviceLibrary.Execute Command    mkdir /data

Collect Logs
    Collect Workflow Logs
    Collect Systemd Logs

Collect Systemd Logs
    Execute Command    sudo journalctl -n 10000

Collect Workflow Logs
    Execute Command    cat /var/log/tedge/agent/*
