FROM golang:1.20-bullseye AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./ ./

RUN go build -o /pg_pro

## Deploy
FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /pg_pro /pg_pro
COPY config.yaml ./
USER nonroot:nonroot

ENTRYPOINT ["/pg_pro"]
