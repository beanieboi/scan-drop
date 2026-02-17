FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o scan-drop ./...

FROM alpine:latest

RUN apk --no-cache add ca-certificates
RUN addgroup -S scanuser && adduser -S -G scanuser scanuser

WORKDIR /app

COPY --from=builder /app/scan-drop .

RUN mkdir -p /opt/paperless/consume && \
    chown -R scanuser:scanuser /opt/paperless/consume && \
    chown scanuser:scanuser /app/scan-drop

USER scanuser

EXPOSE 2121
CMD ["./scan-drop"]
