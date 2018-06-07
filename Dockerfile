FROM gcr.io/distroless/base
EXPOSE 9333
COPY version_exporter /
ENTRYPOINT ["/version_exporter"]
