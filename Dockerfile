ARG REGISTRY=gcr.io
ARG IMAGE=distroless/static:nonroot
FROM ${REGISTRY}/${IMAGE}
ARG GOOS=linux
ARG TARGETARCH
ARG USER=nonroot

ENV TZ="Europe/Moscow"

WORKDIR /
COPY --chmod=755 bin/selebrow-${GOOS}-${TARGETARCH:-amd64} /bin/selebrow
COPY config /config/
USER $USER
EXPOSE 4444
ENTRYPOINT [ "/bin/selebrow" ]
