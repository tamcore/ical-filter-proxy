# syntax=docker/dockerfile:1

# Production image for goreleaser (dockers_v2). The ical-filter-proxy binary is
# pre-built by goreleaser with the IANA tzdata embedded, so no build stage is
# needed here.
FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETPLATFORM

LABEL org.opencontainers.image.source="https://github.com/tamcore/ical-filter-proxy"
LABEL org.opencontainers.image.description="Filtering proxy for remote iCalendar feeds"
LABEL org.opencontainers.image.licenses="MIT"

COPY ${TARGETPLATFORM}/ical-filter-proxy /ical-filter-proxy

EXPOSE 8000

USER 65532:65532

ENTRYPOINT ["/ical-filter-proxy"]
