ARTIFACT_ID=sonarcarp
VERSION=11.1.5-4
MAKEFILES_VERSION=9.2.1

TARGETDIR=target
PKG=${ARTIFACT_ID}-${VERSION}.tar.gz
BINARY=${TARGETDIR}/${ARTIFACT_ID}
GOTAG="1.22"
GO_ENV_VARS=CGO_ENABLED=0 GOOS=linux

include build/make/variables.mk
include build/make/self-update.mk
include build/make/clean.mk
include build/make/dependencies-gomod.mk
include build/make/digital-signature.mk
include build/make/test-unit.mk
include build/make/mocks.mk
include build/make/build.mk

.DEFAULT_GOAL := build-carp

.PHONY: build-carp
build-carp: vendor compile ## Compiles the sonarcarp binary

