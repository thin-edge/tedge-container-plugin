FROM debian:12-slim
ARG TARGETPLATFORM
ARG TEDGE_CHANNEL=main

RUN \
    # Prevent services from auto-starting during package installation
    echo "exit 101" | tee /usr/sbin/policy-rc.d \
    && chmod +x /usr/sbin/policy-rc.d \
    && apt-get -y update --allow-releaseinfo-change \
    && DEBIAN_FRONTEND=noninteractive apt-get -y --no-install-recommends install -o Dpkg::Options::="--force-confnew" \
        locales \
        wget \
        curl \
        gnupg2 \
        sudo \
        apt-transport-https \
        ca-certificates \
        systemd \
        systemd-sysv \
        # shells
        bash \
        bash-completion \
        zsh \
        vim.tiny \
        mosquitto \
        mosquitto-clients \
        # docker
        docker.io \
        docker-compose \
        # fuse-overlayfs is required when running Docker-in-Docker (overlay-on-overlay)
        fuse-overlayfs \
        fuse3 \
    # Remove unnecessary systemd services
    && rm -f /lib/systemd/system/multi-user.target.wants/* \
        /etc/systemd/system/*.wants/* \
        /lib/systemd/system/local-fs.target.wants/* \
        /lib/systemd/system/sockets.target.wants/*udev* \
        /lib/systemd/system/sockets.target.wants/*initctl* \
        /lib/systemd/system/systemd-update-utmp* \
    && mkdir -p /etc/docker \
    && echo '{"storage-driver":"fuse-overlayfs"}' > /etc/docker/daemon.json \
    && systemctl enable docker \
    # restore settings after installation
    && echo "exit 0" | tee /usr/sbin/policy-rc.d

# install and configure thin-edge.io
RUN wget -O - thin-edge.io/install.sh | sh -s -- --channel "${TEDGE_CHANNEL}" \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        tedge-command-plugin \
    && tedge config set mqtt.bind.address 0.0.0.0

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

RUN echo "tedge  ALL = (ALL) NOPASSWD: /usr/bin/tedge-container" | tee /etc/sudoers.d/tedge-containers \
    && mkdir -p /etc/tedge-container-plugin \
    && echo "CONTAINER_METRICS_INTERVAL=60s" | tee -a /etc/tedge-container-plugin/env

ADD https://raw.githubusercontent.com/thin-edge/tedge-demo-container/refs/heads/main/images/common/bootstrap.sh /usr/bin/bootstrap.sh
RUN chmod a+x /usr/bin/bootstrap.sh

# Reference: https://developers.redhat.com/blog/2019/04/24/how-to-run-systemd-in-a-container#enter_podman
# STOPSIGNAL SIGRTMIN+3 (=37)
STOPSIGNAL 37

CMD ["/lib/systemd/systemd"]
