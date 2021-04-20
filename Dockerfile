FROM golang:1.16.3 as builder
ARG PACKAGE=pod-watcher
LABEL builder=true
COPY ${PACKAGE}/ /go/src/
RUN \
  set -x \
  && \
  cd /go/src/ \
  && \
  CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/${PACKAGE} \
  && \
  chown 1001:0 /go/bin/${PACKAGE}

FROM scratch
LABEL \
  maintainer="kin.wai.koo@gmail.com" \
  io.k8s.description="Controller that annotates pods with information of the node that the pod is scheduled on." \
  org.opencontainers.image.source="https://github.com/kwkoo/pod-watcher-controller" \
  builder=false

COPY --from=builder /go/bin/${PACKAGE} /usr/bin/
USER 1001
ENTRYPOINT ["/usr/bin/pod-watcher"]
