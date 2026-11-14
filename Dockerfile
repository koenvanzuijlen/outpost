FROM golang:1.27.4 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

RUN mkdir /data

COPY . .

ENV CGO_ENABLED=0
ARG VERSION=dev
RUN go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/outpost ./cmd/outpost

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /out/outpost /outpost
COPY --from=build --chown=nonroot:nonroot /data /data

ENV DATABASE_PATH=/data/outpost.db

USER nonroot

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/outpost", "healthcheck"]

VOLUME /data

ENTRYPOINT ["/outpost"]
