FROM golang:1.23-alpine as builder

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/dockge-gitops ./cmd/main.go

FROM scratch

COPY --from=builder /go/bin/dockge-gitops /app/cmd/dockge-gitops

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app/cmd
CMD ["/app/cmd/dockge-gitops"]
