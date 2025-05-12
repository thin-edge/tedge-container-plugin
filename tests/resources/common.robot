*** Settings ***
Library    DeviceLibrary    bootstrap_script=bootstrap.sh

*** Variables ***

# Cumulocity settings
&{C8Y_CONFIG}        host=%{C8Y_BASEURL= }    username=%{C8Y_USER= }    password=%{C8Y_PASSWORD= }    tenant=%{C8Y_TENANT= }

# Docker adapter settings (to control which image is used in the system tests).
# The user just needs to set the TEST_IMAGE env variable
&{DOCKER_CONFIG}    image=%{TEST_IMAGE=}

*** Keywords ***

Collect Logs
    Collect Workflow Logs
    Collect Systemd Logs

Collect Systemd Logs
    Execute Command    if command -V journalctl >/dev/null 2>&1; then sudo journalctl -n 10000 | grep -v -e ' kernel: audit:'; else head -n 10000 /var/log/*.log; fi

Collect Workflow Logs
    Execute Command    cat /var/log/tedge/agent/*
