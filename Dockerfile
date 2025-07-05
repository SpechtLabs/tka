FROM alpine:latest

LABEL org.opencontainers.image.title="tailscale-k8s-auth"
LABEL org.opencontainers.image.source="https://github.com/SpechtLabs/tailscale-k8s-auth"
LABEL org.opencontainers.image.description="tailscale-k8s-auth provides secure, ephemeral Kubernetes access using your Tailscale identity and network."
LABEL org.opencontainers.image.licenses="MIT"

COPY ./tailscale-k8s-auth /bin/tailscale-k8s-auth

ENTRYPOINT ["/bin/tailscale-k8s-auth"]
CMD [""]

EXPOSE 8099
EXPOSE 50051

