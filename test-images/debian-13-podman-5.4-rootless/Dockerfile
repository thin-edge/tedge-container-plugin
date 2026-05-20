FROM debian:13-slim
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
        # podman
        dbus-user-session \
        uidmap \
        nftables \
        slirp4netns \
        passt \
        aardvark-dns \
        podman \
        podman-compose \
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
    && mkdir -p /etc/containers/registries.conf.d \
    && echo 'unqualified-search-registries = ["docker.io"]' | tee /etc/containers/registries.conf.d/docker.conf \
    && touch /etc/containers/nodocker \
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
    && echo "CONTAINER_METRICS_INTERVAL=60s" | tee -a /etc/tedge-container-plugin/env \
    # && echo "CONTAINER_METRICS_INTERVAL=60s" | tee -a /etc/tedge-container-plugin/env \
    # Set the container host in the plugin config so all components (service, sm-plugins, etc.)
    # use the rootless podman socket. Resolve the UID at build time since the config is static.
    && sed -i "s|.*#host = .*|host = \"unix:///run/user/$(id -u tedge)/podman/podman.sock\"|" \
        /etc/tedge/plugins/tedge-container-plugin.toml

# Setup podman rootless for the tedge user
RUN usermod -d /home/tedge tedge \
    && mkdir -p /home/tedge/.config/containers \
    && chown -R tedge:tedge /home/tedge \
    && usermod --add-subuids 100000-165535 --add-subgids 100000-165535 tedge \
    # Enable persistent user session at boot (equivalent of: loginctl enable-linger tedge)
    && mkdir -p /var/lib/systemd/linger \
    && touch /var/lib/systemd/linger/tedge \
    # Enable fuse-overlayfs for overlay-on-overlay (Podman-in-Docker) in rootless storage config
    && printf '[storage]\ndriver = "overlay"\n\n[storage.options.overlay]\nmount_program = "/usr/bin/fuse-overlayfs"\n' \
        > /home/tedge/.config/containers/storage.conf \
    # Use cgroupfs manager to avoid requiring a systemd user session / D-Bus.
    # Use file events backend: the default "journald" backend does not deliver
    # events reliably inside a container (the tedge user cannot write/read the
    # user journal), which would make `podman events` and the plugin event
    # stream silently produce nothing.
    # Use k8s-file log driver: the default "journald" log driver also fails
    # for rootless users inside a container (no access to the user journal),
    # which would make `podman logs` return empty output or an error.
    && printf '[containers]\nlog_driver = "k8s-file"\n\n[engine]\ncgroup_manager = "cgroupfs"\nevents_logger = "file"\n' \
        > /home/tedge/.config/containers/containers.conf

# Create a system-level service to start the rootless podman socket for the tedge user.
# User-level socket units (podman.socket) require a systemd user session which is
# unreliable in Docker containers. A system service with User=tedge is more robust.
# The UID is baked in at build time to avoid systemd specifier expansion issues.
RUN TEDGE_UID=$(id -u tedge) \
    && { \
        echo "[Unit]"; \
        echo "Description=Podman API socket for tedge user"; \
        echo "# user-runtime-dir@.service is managed by systemd-logind and mounts the"; \
        echo "# tmpfs at /run/user/UID. Declaring it as a hard dependency prevents a race"; \
        echo "# where logind mounts a fresh tmpfs after ExecStartPre created the podman"; \
        echo "# subdirectory, which would silently wipe the socket."; \
        echo "Requires=user-runtime-dir@${TEDGE_UID}.service"; \
        echo "After=multi-user.target user-runtime-dir@${TEDGE_UID}.service"; \
        echo ""; \
        echo "[Service]"; \
        echo "Type=simple"; \
        echo "User=tedge"; \
        echo "Environment=XDG_RUNTIME_DIR=/run/user/${TEDGE_UID}"; \
        echo "ExecStartPre=+/bin/install -d -m 0700 -o tedge -g tedge /run/user/${TEDGE_UID} /run/user/${TEDGE_UID}/podman"; \
        echo "ExecStart=/usr/bin/podman system service --time=0 unix:///run/user/${TEDGE_UID}/podman/podman.sock"; \
        echo "Restart=on-failure"; \
        echo "RestartSec=5"; \
        echo ""; \
        echo "[Install]"; \
        echo "WantedBy=multi-user.target"; \
    } > /etc/systemd/system/podman-tedge.service \
    && systemctl enable podman-tedge.service

# Run tedge-container-plugin as the tedge user.
RUN mkdir -p /etc/systemd/system/tedge-container-plugin.service.d \
    && printf '[Service]\nUser=tedge\n' \
        > /etc/systemd/system/tedge-container-plugin.service.d/rootless.conf

ADD https://raw.githubusercontent.com/thin-edge/tedge-demo-container/refs/heads/main/images/common/bootstrap.sh /usr/bin/bootstrap.sh
RUN chmod a+x /usr/bin/bootstrap.sh

# Reference: https://developers.redhat.com/blog/2019/04/24/how-to-run-systemd-in-a-container#enter_podman
# STOPSIGNAL SIGRTMIN+3 (=37)
STOPSIGNAL 37

CMD ["/lib/systemd/systemd"]
