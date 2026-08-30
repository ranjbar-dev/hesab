FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 go build -o /out/seed ./cmd/seed

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/server /usr/local/bin/server
COPY --from=build /out/seed /usr/local/bin/seed
EXPOSE 8080
ENTRYPOINT ["server"]
