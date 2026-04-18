FROM gcr.io/distroless/static:nonroot
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/gantry /usr/local/bin/gantry
ENTRYPOINT ["/usr/local/bin/gantry"]
