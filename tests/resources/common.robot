*** Variables ***

# Cumulocity settings
&{C8Y_CONFIG}        host=%{C8Y_BASEURL= }    username=%{C8Y_USER= }    password=%{C8Y_PASSWORD= }    tenant=%{C8Y_TENANT= }

# Docker adapter settings (to control which image is used in the system tests).
# The user just needs to set the IMAGE env variable
&{DOCKER_CONFIG}    image=%{IMAGE=}
