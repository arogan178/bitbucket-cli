# GoReleaser consumes this Dockerfile; it expects `bt` to already be
# built and copied into the build context alongside LICENSE and README.
FROM gcr.io/distroless/static-debian12:nonroot

COPY bt /usr/local/bin/bt
COPY LICENSE /LICENSE
COPY README.md /README.md

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/bt"]
