FROM alpine:latest

ARG TARGETPLATFORM
LABEL org.opencontainers.image.title="tailscale-k8s-auth"
LABEL org.opencontainers.image.source="https://github.com/spechtlabs/tka"
LABEL org.opencontainers.image.description="tka provides secure, ephemeral Kubernetes access using your Tailscale identity and network."
LABEL org.opencontainers.image.licenses="Apache 2.0"

COPY $TARGETPLATFORM/ts-k8s-srv /bin/ts-k8s-srv

ENTRYPOINT ["/bin/ts-k8s-srv"]
CMD [""]

EXPOSE 8099
EXPOSE 50051
