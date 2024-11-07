FROM ghcr.io/thin-edge/tedge-demo-main-systemd:latest
ARG TARGETPLATFORM

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y \
        # Install with all recommended packages as this is simplier to maintain
        podman \
    && DEBIAN_FRONTEND=noninteractive apt-get install -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confnew" -y --no-install-recommends \
        podman-compose \
    && tedge config unset c8y.proxy.client.host \
    && tedge config unset mqtt.client.host \
    && tedge config unset http.client.host

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
    # FIXME: Replace once project is merged back with tedge-container-plugin
    # && apt-get install -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confnew" --allow-downgrades -y /tmp/*${PKG_ARCH}*.deb \
    && dpkg -i --force-overwrite /tmp/*${PKG_ARCH}*.deb \
    && rm -f /tmp/*.deb
