export PROJECT_ROOT := $(realpath .)
export IMAGE_REPOSITORY ?= ghcr.io/aixcyberchallenge/competition-api
export IMAGE_TAGS ?= latest
export IMAGE_LABELS ?=

# change github images.yaml workflow if this list changes
CONTAINER_DIRS := competition-api/cmd/mock_server competition-api/cmd/mock_crs competition-api/cmd/mock_jobrunner competition-api dind

.PHONY: podman-build podman-push container-dirs should-build

container-dirs:
	@echo ${CONTAINER_DIRS}

should-build:
	for dir in ${CONTAINER_DIRS}; do \
		if $(MAKE) -C $$dir should-build; then exit 0; fi; \
	done; \
	exit 1

podman-build:
	for dir in ${CONTAINER_DIRS}; do \
	   $(MAKE) -C $$dir podman-build ; \
	done

podman-push:
	for dir in ${CONTAINER_DIRS}; do \
	   $(MAKE) -C $$dir podman-push ; \
	done
