FROM golang:1.27.3 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

RUN mkdir /data

COPY . .

ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/outpost ./cmd/outpost

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /out/outpost /outpost
COPY --from=build --chown=nonroot:nonroot /data /data

ENV DATABASE_PATH=/data/outpost.db

USER nonroot

EXPOSE 8080

VOLUME /data

ENTRYPOINT ["/outpost"]
