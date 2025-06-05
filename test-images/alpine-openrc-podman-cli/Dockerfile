FROM docker.io/alpine:3.20
ARG TARGETPLATFORM

RUN apk add --no-cache \
        openrc \
        mdevd-openrc \
        wget \
        curl \
        bash \
        sudo \
        jq \
        dasel \
        podman \
        podman-compose \
        # Required when running inside docker
        # see https://github.com/containers/buildah/issues/3666
        fuse-overlayfs \
    && wget -O - thin-edge.io/install.sh | sh -s \
    && apk add \
        tedge-apk-plugin \
        tedge-command-plugin

ADD https://raw.githubusercontent.com/thin-edge/tedge-demo-container/refs/heads/main/images/common/bootstrap.sh /usr/bin/
RUN chmod 755 /usr/bin/bootstrap.sh

RUN sed -i '/getty/d' /etc/inittab \
    && sed -i 's/#mount_program/mount_program/' /etc/containers/storage.conf

COPY dist/*.apk /tmp/
RUN case ${TARGETPLATFORM} in \
        "linux/386")  PKG_ARCH=linux_386  ;; \
        "linux/amd64")  PKG_ARCH=linux_amd64  ;; \
        "linux/arm64")  PKG_ARCH=linux_arm64  ;; \
        "linux/arm/v6")  PKG_ARCH=linux_armv6  ;; \
        "linux/arm/v7")  PKG_ARCH=linux_armv7  ;; \
        *) echo "Unsupported target platform: TARGETPLATFORM=$TARGETPLATFORM"; exit 1 ;; \
    esac \
    && apk add --allow-untrusted /tmp/*${PKG_ARCH}*.apk \
    && mkdir -p /opt/packages \
    && cp /tmp/*${PKG_ARCH}*.apk /opt/packages/ \
    && rm -f /tmp/*.apk

RUN echo "tedge  ALL = (ALL) NOPASSWD: /usr/bin/tedge-container" | tee /etc/sudoers.d/tedge-containers \
    && mkdir -p /etc/tedge-container-plugin \
    && dasel put -r toml -t string -v '60s' 'metrics.interval' --indent 0 < /etc/tedge/plugins/tedge-container-plugin.toml > /etc/tedge/plugins/tedge-container-plugin.toml.tmp \
    && mv /etc/tedge/plugins/tedge-container-plugin.toml.tmp /etc/tedge/plugins/tedge-container-plugin.toml

# Default services
RUN rc-update add podman

CMD ["/sbin/init"]
