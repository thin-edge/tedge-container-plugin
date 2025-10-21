FROM ghcr.io/thin-edge/tedge-demo-main-systemd:latest
ARG TARGETPLATFORM

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y \
        # Install with all recommended packages as this is simplier to maintain
        && install -m 0755 -d /etc/apt/keyrings \
        && curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc \
        && chmod a+r /etc/apt/keyrings/docker.asc \    
        && echo \
            "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian \
            $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
        && apt-get update -y \
    && DEBIAN_FRONTEND=noninteractive apt-get install -o Dpkg::Options::="--force-confnew" -y --no-install-recommends \
        # Installing docker-ce at build time fails.
        # See https://github.com/docker/cli/issues/4807 for details
        # docker-ce \
        # Docker cli
        docker-ce-cli \
        docker-compose-plugin \
    && tedge config unset c8y.proxy.client.host \
    && tedge config unset mqtt.client.host \
    && tedge config unset http.client.host

# install from main channel
RUN wget -O - thin-edge.io/install.sh | sh -s -- --channel main

RUN echo "tedge  ALL = (ALL) NOPASSWD: /usr/bin/tedge-container" | tee /etc/sudoers.d/tedge-containers \
    && echo "CONTAINER_METRICS_INTERVAL=60s" | tee -a /etc/tedge-container-plugin/env

COPY dist/*.deb /tmp/
RUN case ${TARGETPLATFORM} in \
        "linux/386")  PKG_ARCH=linux_386  ;; \
        "linux/amd64")  PKG_ARCH=linux_amd64  ;; \
        "linux/arm64")  PKG_ARCH=linux_arm64  ;; \
        "linux/arm/v6")  PKG_ARCH=linux_armv6  ;; \
        "linux/arm/v7")  PKG_ARCH=linux_armv7  ;; \
        *) echo "Unsupported target platform: TARGETPLATFORM=$TARGETPLATFORM"; exit 1 ;; \
    esac \
    && apt-get install -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confnew" --allow-downgrades -y /tmp/*${PKG_ARCH}*.deb \
    && mkdir -p /opt/packages \
    && cp /tmp/*${PKG_ARCH}*.deb /opt/packages/ \
    && rm -f /tmp/*.deb
