# GoReleaser consumes this Dockerfile; it expects `bt` to already be
# built and copied into the build context.
FROM gcr.io/distroless/static-debian12:nonroot

COPY bt /usr/local/bin/bt

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/bt"]
