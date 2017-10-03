FROM scratch
EXPOSE 9222
WORKDIR /
COPY version_exporter .
ENTRYPOINT ["./version_exporter"]
