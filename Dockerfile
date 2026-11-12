FROM golang:1.27.4-alpine@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS build
ENV GOTOOLCHAIN=local
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG REVISION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.Rev=${REVISION} -X main.deploymentProfile=vibenest" -o /csp_collector .

FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
LABEL maintainer="https://github.com/jacobbednarz/go-csp-collector"
COPY --from=build /csp_collector /
EXPOSE 8080

RUN addgroup -Sg 1000 csp_collector && \
    adduser -SG csp_collector -u 1000 -h /home/csp_collector csp_collector

# A numeric user lets Kubernetes verify runAsNonRoot without a runAsUser.
USER 1000:1000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s CMD ["wget", "-q", "-O", "/dev/null", "http://127.0.0.1:8080/_healthcheck"]
ENTRYPOINT ["/csp_collector"]
