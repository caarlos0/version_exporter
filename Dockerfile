FROM scratch
EXPOSE 9333
WORKDIR /
COPY version_exporter .
ENTRYPOINT ["./version_exporter"]
