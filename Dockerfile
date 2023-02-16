FROM cgr.dev/chainguard/static
EXPOSE 9333
COPY version_exporter /
COPY config.yaml /
ENTRYPOINT ["/version_exporter"]
