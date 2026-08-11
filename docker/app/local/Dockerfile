FROM golang:1.26.1

RUN go install github.com/cespare/reflex@v0.3.1 && \
    go install github.com/go-delve/delve/cmd/dlv@latest

COPY reflex.conf /

ENTRYPOINT ["reflex", "-c", "/reflex.conf"]