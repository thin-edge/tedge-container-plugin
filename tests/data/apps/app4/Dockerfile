FROM ghcr.io/thin-edge/tedge-container-bundle:20241123.1853

USER root
RUN apk del \
        docker-cli \
        docker-cli-compose \
    && apk add --no-cache \
        podman-remote

ENV REGISTRY_AUTH_FILE=/run/containers/0/auth.json
USER tedge
CMD [ "sleep", "infinity" ]
