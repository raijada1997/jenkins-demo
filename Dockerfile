ARG BUILDPLATFORM
ARG TARGETPLATFORM

# ---------- Build Stage ----------
FROM --platform=${BUILDPLATFORM} docker-na-public.artifactory.swg-devops.com/hyc-abell-devops-team-dev-docker-local/isf-stack/golang:isf-v2.13.0 as builder

ARG TARGETOS
ARG TARGETARCH
ENV TARGETOS=${TARGETOS:-linux}
ENV TARGETARCH=${TARGETARCH:-amd64}

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY controllers/ controllers/
COPY elastic/ elastic/
COPY models/ models/
COPY routers/ routers/
COPY services/ services/
COPY main.go main.go

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o dashboard-demo

# ---------- Runtime Stage ----------
FROM --platform=${TARGETPLATFORM} docker-na-public.artifactory.swg-devops.com/hyc-abell-devops-team-dev-docker-local/isf-stack/ubi9-minimal:isf-v2.13.0
WORKDIR /app

RUN microdnf install -y ca-certificates && microdnf clean all

COPY --from=builder /workspace/dashboard-demo .

USER 1001

EXPOSE 8080

ENTRYPOINT ["./dashboard-demo"]
