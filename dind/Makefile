IMAGE_NAME := docker
BUILD_DEPENDENCIES := images.tar.gz

VERSIONS_TO_CACHE := v1.3.0

# make skips the if check without an unconditional use of the variable outside of the if condition...
$(eval dummy := $(PROJECT_ROOT))
ifdef PROJECT_ROOT
	SOURCES := $(shell find ${PROJECT_ROOT}/dind -type f) ${PROJECT_ROOT}/.github/workflows/images.yaml ${PROJECT_ROOT}/Makefile ${PROJECT_ROOT}/container.mk
	include ${PROJECT_ROOT}/container.mk
endif

images.tar.gz:
	set -ex; \
	to_archive=""; \
	for version in ${VERSIONS_TO_CACHE}; do \
		images="ghcr.io/aixcyberchallenge/base-runner:$${version} ghcr.io/aixcyberchallenge/base-runner-debug:$${version} ghcr.io/aixcyberchallenge/base-image:$${version} ghcr.io/aixcyberchallenge/base-builder:$${version} ghcr.io/aixcyberchallenge/base-clang:$${version} ghcr.io/aixcyberchallenge/base-builder-jvm:$${version}"; \
		for image in $${images}; do \
			podman pull "$${image}"; \
			to_archive="$${to_archive} $${image}"; \
		done; \
	done; \
	podman save -m -o ${PROJECT_ROOT}/dind/images.tar.gz $${to_archive}
