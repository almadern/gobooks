FROM debian:stable
RUN apt-get update && apt-get upgrade -y && apt-get install -y ca-certificates
COPY ./gobooks /usr/bin/
COPY converter /opt/converter
COPY html /opt/html
ENV HTML_PATH='/opt/html'
ENV CONVERTER_PATH='/opt/converter'
ENV CGO_ENABLED=1
CMD ["/bin/sh", "-c", "/usr/bin/gobooks"]
