FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/5gcore-nrf ./cmd/nrf && \
    go build -o /out/5gcore-amf ./cmd/amf && \
    go build -o /out/5gcore-smf ./cmd/smf && \
    go build -o /out/5gcore-ausf ./cmd/ausf && \
    go build -o /out/5gcore-upf ./cmd/upf && \
    go build -o /out/5gcore-udm ./cmd/udm && \
    go build -o /out/5gcore-ue-sim ./cmd/ue-sim

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/5gcore-nrf /app/5gcore-nrf
COPY --from=builder /out/5gcore-amf /app/5gcore-amf
COPY --from=builder /out/5gcore-smf /app/5gcore-smf
COPY --from=builder /out/5gcore-ausf /app/5gcore-ausf
COPY --from=builder /out/5gcore-upf /app/5gcore-upf
COPY --from=builder /out/5gcore-udm /app/5gcore-udm
COPY --from=builder /out/5gcore-ue-sim /app/5gcore-ue-sim
CMD ["/bin/sh"]
