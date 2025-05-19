# syntax=docker/dockerfile:1

ARG GO_VERSION=1.24
ARG ALPINE_VERSION=3.21

ARG DOCS_FORMATS="md,yaml"

FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS base
RUN apk add --no-cache rsync git
ENV GOFLAGS=-mod=vendor
ENV CGO_ENABLED=0
WORKDIR /src

FROM base AS docs-gen
WORKDIR /src
RUN --mount=target=. \
    --mount=target=/root/.cache,type=cache \
    go build -mod=vendor -o /out/docsgen ./docs/generate.go

FROM base AS docs-build
COPY --from=docs-gen /out/docsgen /usr/bin
ARG DOCS_FORMATS
RUN --mount=target=/context \
    --mount=target=.,type=tmpfs <<EOT
  set -e
  rsync -a /context/. .
  docsgen --formats "$DOCS_FORMATS" --source "docs/reference"
  mkdir /out
  cp -r docs/reference/* /out/
EOT

FROM scratch AS docs-update
COPY --from=docs-build /out /

FROM docs-build AS docs-validate
RUN --mount=target=/context \
    --mount=target=.,type=tmpfs <<EOT
  set -e
  rsync -a /context/. .
  git add -A
  rm -rf docs/reference/*
  cp -rf /out/* ./docs/reference/
  if [ -n "$(git status --porcelain -- docs/reference)" ]; then
    echo >&2 'ERROR: Docs result differs. Please update with "make docs"'
    git status --porcelain -- docs/reference
    exit 1
  fi
EOT
