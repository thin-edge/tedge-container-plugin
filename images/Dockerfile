FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY bin/tedge-container /usr/bin
ENTRYPOINT [ "/usr/bin/tedge-container" ]

ENV CONTAINER_LOG_LEVEL=info

ENV CONTAINER_FILTER_INCLUDE_IDS=
ENV CONTAINER_FILTER_INCLUDE_LABELS=
ENV CONTAINER_FILTER_INCLUDE_NAMES=
ENV CONTAINER_FILTER_INCLUDE_TYPES=
ENV CONTAINER_FILTER_EXCLUDE_LABELS=
ENV CONTAINER_FILTER_EXCLUDE_NAMES=

ENV CONTAINER_DELETE_FROM_CLOUD_ENABLED=
ENV CONTAINER_EVENTS_ENABLED=
ENV CONTAINER_METRICS_ENABLED=
ENV CONTAINER_METRICS_INTERVAL=

ENV CONTAINER_TOPIC_ID=
ENV CONTAINER_TOPIC_ROOT=
ENV CONTAINER_SERVICE_NAME=

ENV CONTAINER_DEVICE_ID=

# TLS
ENV CONTAINER_CLIENT_KEY=
ENV CONTAINER_CLIENT_CERT_FILE=
ENV CONTAINER_CLIENT_CA_FILE=

CMD ["run"]