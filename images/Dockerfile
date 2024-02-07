FROM ghcr.io/thin-edge/tedge-demo-main-systemd:latest

RUN install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc \
    && sudo chmod a+r /etc/apt/keyrings/docker.asc \
    # Add the repository to Apt sources:
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian \
        $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
        sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        docker-ce-cli \
        docker-compose-plugin
    # && groupadd docker \
    # && usermod -aG docker

COPY ./images/container-env /etc/tedge-container-plugin/env

COPY dist/tedge-container-plugin*.deb /tmp/
RUN dpkg -i /tmp/*.deb
