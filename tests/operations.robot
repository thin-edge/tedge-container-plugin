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

Install/uninstall container package
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker rm -f webserver; sleep 1
    ${operation}=    Cumulocity.Install Software    {"name": "webserver", "version": "ghcr.io/thin-edge/test-images/httpd:2.4", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Device Should Have Installed Software    {"name": "webserver", "version": "ghcr.io/thin-edge/test-images/httpd:2.4", "softwareType": "container"}
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker run --rm -t --network tedge ghcr.io/thin-edge/test-images/busybox wget -O- webserver:80;
    Operation Should Be SUCCESSFUL    ${operation}
    Should Contain    ${operation.to_json()["c8y_Command"]["result"]}    It works!
    Cumulocity.Should Have Services    name=webserver    service_type=container    status=up

    # Check if you can request the logs for it
    Cumulocity.Should Contain Supported Log Types    webserver::container
    ${operation}=    Cumulocity.Get Log File    webserver::container
    Operation Should Be SUCCESSFUL    ${operation}

    # Uninstall
    ${operation}=     Cumulocity.Uninstall Software    {"name": "webserver", "version": "ghcr.io/thin-edge/test-images/httpd:2.4", "softwareType": "container"}
    Operation Should Be SUCCESSFUL    ${operation}
    Device Should Not Have Installed Software    webserver
    Cumulocity.Should Have Services    name=webserver    service_type=container    min_count=0    max_count=0

    # container's log type should be removed
    Cumulocity.Should Not Contain Supported Log Types    webserver::container

Install/uninstall container package from file
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker rm -f app3; sleep 1
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
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker rm -f manualapp1; sleep 1
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker network create tedge ||: ; sudo tedge-container engine docker run -d --network tedge --name manualapp1 ghcr.io/thin-edge/test-images/httpd:2.4
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60

    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker run --rm -t --network tedge ghcr.io/thin-edge/test-images/busybox wget -O- manualapp1:80;
    Operation Should Be SUCCESSFUL    ${operation}

    Should Contain    ${operation.to_json()["c8y_Command"]["result"]}    It works!
    Cumulocity.Should Have Services    name=manualapp1    service_type=container    status=up

    # container's log type should of been added
    Cumulocity.Should Contain Supported Log Types    manualapp1::container

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

    # container's log type should be removed
    Cumulocity.Should Not Contain Supported Log Types    manualapp1::container

Manual container creation/deletion with error on run
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker rm -f manualapp2; sleep 1
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker run -d --name manualapp2 ghcr.io/thin-edge/test-images/httpd:2.4 --invalid-arg || exit 0
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Cumulocity.Should Have Services    name=manualapp2    service_type=container    status=down

    # Uninstall
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker rm manualapp2 --force
    Operation Should Be SUCCESSFUL    ${operation}
    Cumulocity.Should Have Services    name=manualapp2    service_type=container    min_count=0    max_count=0    timeout=10


Manual container created and then killed
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker rm -f manualapp3; sleep 1
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker run -d --name manualapp3 ghcr.io/thin-edge/test-images/busybox sh -c 'exec sleep infinity'
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
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker rm -f manualapp4; sleep 1
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker run -d --name manualapp4 ghcr.io/thin-edge/test-images/busybox sh -c 'exec sleep infinity'
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
    Cumulocity.Should Have Services    name=manualapp4    service_type=container

    # Start the service, and check that the service has been removed (without the explicit service type defined)
    Start Service    tedge-container-plugin
    Cumulocity.Should Have Services    name=manualapp4    min_count=0    max_count=0    timeout=10

Remove Orphaned Cloud Services eventually if Cumulocity Proxy is Unavailable at deletion time
    [Documentation]    Some instances the Cumulocity local proxy will not be available
    ...    so the syncing of the services in the cloud should be able to handle
    ...    a delayed sync when the Cumulocity local proxy is unavailable, or the device
    ...    the device went offline at the time of installation or removal of a container
    ...    or container-group.
    ...    See https://github.com/thin-edge/tedge-container-plugin/issues/181
    ...
    ...    The delayed sync is recovered by the bridge-online handler (when the mapper
    ...    health message arrives) or by the failure-driven retry (reconcile.retry_interval,
    ...    default 30s) when that message is lost (thin-edge/thin-edge.io#3185), so the
    ...    cleanup can legitimately take up to ~60s after the mapper is back online.
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker rm -f manualapp5; sleep 1

    # create a local container manually
    ${operation}=    Cumulocity.Execute Shell Command    sudo tedge-container engine docker run -d --name manualapp5 ghcr.io/thin-edge/test-images/busybox sh -c 'exec sleep infinity'
    Operation Should Be SUCCESSFUL    ${operation}    timeout=60
    Cumulocity.Should Have Services    name=manualapp5    service_type=container    status=up

    # install a container-group
    Install container-group application    app6    1.0.0    app5    ${CURDIR}/data/apps/app5.tar.gz
    Device Should Have Installed Software    {"name": "app6", "version": "1.0.0", "softwareType": "container-group"}
    Cumulocity.Should Have Services    name=app6@httpd    service_type=container-group    status=up

    Stop Service    tedge-mapper-c8y

    # Remove the container (manually)
    DeviceLibrary.Execute Command    cmd=sudo tedge-container engine docker rm manualapp5 --force

    # Remove the container-group (manually as the mapper is down)
    DeviceLibrary.Execute Command    cmd=sudo /etc/tedge/sm-plugins/container-group remove app6 --module-version 1.0.0

    # Start the service, and check that the service has been removed (without the explicit service type defined)
    Sleep    15s
    Start Service    tedge-mapper-c8y

    # Services should be eventually deleted. The timeout must cover a full
    # failure-driven retry cycle (the cleanup is not necessarily triggered by
    # the mapper health message, which can be lost) plus the per-service
    # deletion time right after the mapper starts (~5s each while the proxy
    # and cloud connection warm up).
    Cumulocity.Should Have Services    name=manualapp5    min_count=0    max_count=0    timeout=120
    Cumulocity.Should Have Services    name=app6@httpd    min_count=0    max_count=0    timeout=120

Remove orphaned cloud services via periodic reconciliation when bridge health messages are lost
    [Documentation]    Deterministic reproduction of the root cause of
    ...    https://github.com/thin-edge/tedge-container-plugin/issues/181
    ...
    ...    A container-group is removed while tedge-mapper-c8y (and therefore the local
    ...    Cumulocity proxy) is down, so the plugin's cloud service deletion fails and
    ...    is left pending. Normally the bridge-online handler retries it as soon as the
    ...    mapper's health message arrives, however that message can be lost in the field
    ...    (retained message loss in mosquitto 2.0.11-2.2.0, thin-edge/thin-edge.io#3185),
    ...    leaving the service in the cloud forever with an "Unknown" status.
    ...
    ...    The message loss is simulated deterministically: the plugin is suspended
    ...    (SIGSTOP) and mosquitto is restarted so the plugin's MQTT session is dropped
    ...    while it cannot react. The mapper is started while the plugin is disconnected,
    ...    and the retained bridge health messages are cleared before the plugin is
    ...    resumed. The plugin then reconnects and resubscribes, but no bridge-online
    ...    health message exists or arrives, so the orphaned cloud service can only be
    ...    cleaned up by the trigger-independent mechanisms: the failure-driven sync
    ...    retry (reconcile.retry_interval) and the periodic reconcile loop
    ...    (reconcile.interval).

    # Speed up the background reconcile loop (60s is the allowed minimum). Restart to
    # apply it before the orphan is created, so the startup-time update does not
    # interfere with the test. Builds without the reconcile loop ignore this setting.
    DeviceLibrary.Execute Command    cmd=echo 'CONTAINER_RECONCILE_INTERVAL=60s' | sudo tee -a /etc/tedge-container-plugin/env
    DeviceLibrary.Execute Command    cmd=sudo systemctl restart tedge-container-plugin

    # Install a container-group via the cloud (mapper is up)
    Install container-group application    app10    1.0.0    app10    ${CURDIR}/data/apps/app5.tar.gz
    Device Should Have Installed Software    {"name": "app10", "version": "1.0.0", "softwareType": "container-group"}
    Cumulocity.Should Have Services    name=app10@httpd    service_type=container-group    status=up

    # Stop the mapper so the local Cumulocity proxy is unavailable, then remove
    # the container-group locally. The plugin reacts to the container removal
    # events and attempts to delete the cloud service, which fails because the
    # proxy is down, leaving the deletion pending (the entity intentionally stays
    # in the thin-edge.io entity store so it can be retried).
    Stop Service    tedge-mapper-c8y
    DeviceLibrary.Execute Command    cmd=sudo /etc/tedge/sm-plugins/container-group remove app10 --module-version 1.0.0
    Wait Until Keyword Succeeds    30x    2s    Container Group Should Be Removed Locally    app10
    Sleep    10s    reason=Let the plugin process the removal events and fail the pending cloud deletion

    # Suspend the plugin, then restart mosquitto so the plugin's MQTT session is
    # dropped while it cannot react: everything published between now and the
    # moment it resumes and reconnects is invisible to it.
    DeviceLibrary.Execute Command    cmd=sudo systemctl kill --signal=STOP --kill-who=main tedge-container-plugin
    DeviceLibrary.Execute Command    cmd=sudo systemctl restart mosquitto

    # Bring the mapper back online while the plugin is suspended/disconnected, then
    # clear the retained bridge health messages, simulating the retained message
    # loss of mosquitto 2.0.11-2.2.0 (thin-edge/thin-edge.io#3185)
    Start Service    tedge-mapper-c8y
    Sleep    10s    reason=Allow the mapper to connect and publish its health messages
    DeviceLibrary.Execute Command    cmd=for name in tedge-mapper-c8y tedge-mapper-bridge-c8y mosquitto-c8y-bridge; do sudo tedge mqtt pub --retain --qos 1 "te/device/main/service/$name/status/health" ''; done

    # Resume the plugin: it reconnects and resubscribes, but no bridge-online
    # health message exists or arrives, so the bridge-online handler never fires
    DeviceLibrary.Execute Command    cmd=sudo systemctl kill --signal=CONT --kill-who=main tedge-container-plugin

    # Only the trigger-independent mechanisms can clean up the orphan now
    Cumulocity.Should Have Services    name=app10@httpd    min_count=0    max_count=0    timeout=180

    # Confirm the trigger-independent recovery mechanisms ran: the failed cloud
    # deletion must have scheduled a failure-driven retry, and the periodic
    # reconcile loop must be ticking
    DeviceLibrary.Execute Command    cmd=sudo journalctl -u tedge-container-plugin -n 5000 | grep "Scheduling retry of pending cloud sync"
    DeviceLibrary.Execute Command    cmd=sudo journalctl -u tedge-container-plugin -n 5000 | grep "Reconciling container state"

*** Keywords ***

Container Group Should Be Removed Locally
    [Documentation]    Device-local check that a container-group module is no longer
    ...    installed, using the container-group sm-plugin list interface. Used while
    ...    the Cumulocity mapper is stopped, when the cloud software list cannot update.
    [Arguments]    ${name}
    ${output}=    DeviceLibrary.Execute Command    cmd=sudo /etc/tedge/sm-plugins/container-group list    strip=${True}
    Should Not Contain    ${output}    ${name}

Suite Setup
    ${DEVICE_SN}=    Setup
    Set Suite Variable    $DEVICE_SN
    Cumulocity.External Identity Should Exist    ${DEVICE_SN}
    Cumulocity.Should Have Services    name=tedge-container-plugin    service_type=service    min_count=1    max_count=1    timeout=30

    # Create common network for all containers
    ${operation}=    Cumulocity.Execute Shell Command    set -a; . /etc/tedge-container-plugin/env; docker network create tedge ||:

    # Create data directory
    DeviceLibrary.Execute Command    mkdir /data

Install container-group application
    [Documentation]    Install a container-group and let the user do follow up tests
    [Arguments]    ${package_name}    ${package_version}    ${service_name}    ${file}
    ${binary_url}=    Cumulocity.Create Inventory Binary    ${package_name}    container-group    file=${file}
    ${operation}=    Cumulocity.Install Software    {"name": "${package_name}", "version": "${package_version}", "softwareType": "container-group", "url": "${binary_url}"}
    Operation Should Be SUCCESSFUL    ${operation}    timeout=300
