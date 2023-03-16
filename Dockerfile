FROM golang:1.19 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY controllers/ controllers/
COPY addons/ addons/
COPY api/ api/

# Build multicluster manager and add-ons
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o install-addon addons/cmd/install/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o status-sync-addon addons/cmd/status_sync/main.go

FROM alpine:latest

WORKDIR /
RUN apk add libc6-compat
COPY --from=builder /workspace/manager /workspace/manager ./
COPY --from=builder /workspace/install-addon /workspace/install-addon ./
COPY --from=builder /workspace/status-sync-addon /workspace/status-sync-addon ./
USER 65532:65532
