*** Settings ***
Resource    ./resources/common.robot
Library    Cumulocity
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

Suite Setup    Suite Setup
Test Teardown    Collect Logs

Test Tags    docker    podman

*** Variables ***

${IMAGE}    ghcr.io/thin-edge/test-images/httpd:2.4.64

*** Test Cases ***

Install Image From A Registry
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker rmi ${IMAGE} ||:
    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools image-install --image ${IMAGE}
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker image inspect ${IMAGE}    exp_exit_code=0

Install Image From A File
    Save Image To File
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker rmi ${IMAGE}
    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools image-install --image ${IMAGE} --file /tmp/image.tar
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker image inspect ${IMAGE}    exp_exit_code=0

Install Image From A File Which Does Not Contain It
    Save Image To File
    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools image-install --image example.com/other:1.0 --file /tmp/image.tar    exp_exit_code=!0
    # the requested reference must not be left pointing at whatever the file contained
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker image inspect example.com/other:1.0    exp_exit_code=!0

Install Image Without A File Falls Back To The Registry
    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools image-install --image ${IMAGE} --file ""
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker image inspect ${IMAGE}    exp_exit_code=0

*** Keywords ***

Save Image To File
    DeviceLibrary.Execute Command    cmd=sudo tedge-container tools image-install --image ${IMAGE}
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker save ${IMAGE} -o /tmp/image.tar

Suite Setup
    ${DEVICE_SN}=    Setup
    Set Suite Variable    $DEVICE_SN
    Cumulocity.External Identity Should Exist    ${DEVICE_SN}
    Cumulocity.Should Have Services    name=tedge-container-plugin    service_type=service    min_count=1    max_count=1    timeout=30
