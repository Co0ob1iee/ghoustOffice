FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /bin/coreapi ./cmd/coreapi

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /bin/coreapi /app/coreapi
EXPOSE 8081
ENTRYPOINT ["/app/coreapi"]
