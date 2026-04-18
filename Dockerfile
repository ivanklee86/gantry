FROM gcr.io/distroless/static:nonroot
COPY gantry /usr/local/bin/gantry
ENTRYPOINT ["/usr/local/bin/gantry"]
