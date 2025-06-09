ARG REGISTRY=gcr.io
ARG IMAGE=distroless/static:nonroot
FROM ${REGISTRY}/${IMAGE}
ARG GOOS=linux
ARG GOARCH=amd64
ARG USER=nonroot

ENV TZ="Europe/Moscow"

WORKDIR /
COPY bin/selebrow-${GOOS}-${GOARCH} /bin/selebrow
COPY config /config/
USER $USER
EXPOSE 4444
ENTRYPOINT [ "/bin/selebrow" ]
