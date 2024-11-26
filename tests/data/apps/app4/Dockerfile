FROM ghcr.io/thin-edge/tedge-container-bundle:20241123.1853

USER root
RUN apk del \
        docker-cli \
        docker-cli-compose \
    && apk add --no-cache \
        podman-remote

USER tedge
CMD [ "sleep", "infinity" ]
