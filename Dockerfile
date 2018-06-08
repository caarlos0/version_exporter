FROM gcr.io/distroless/base
EXPOSE 9333
COPY version_exporter /
COPY config.yaml /
ENTRYPOINT ["/version_exporter"]
