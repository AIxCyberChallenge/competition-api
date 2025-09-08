.PHONY: podman-build podman-push should-build

_BUILD_TAG_FRAGMENT=$(patsubst %,-t ${IMAGE_REPOSITORY}/${IMAGE_NAME}:%,${IMAGE_TAGS})
_BUILD_LABEL_FRAGMENT=$(foreach label,${IMAGE_LABELS},--label ${label})

should-build:
	set -e -x;
	a=$$(echo "${SOURCES}" | xargs -n1 | jq -R . | jq -s .); \
	b=$$(echo '${CHANGED_FILES}'); \
	count=$$(jq -n --argjson a "$$a" --argjson b "$$b" '$$a - ($$a - $$b) | length'); \
	if [ "$$count" -eq 0 ]; then exit 1; else exit 0; fi

podman-build: ${BUILD_DEPENDENCIES}
	podman build -f Containerfile ${_BUILD_TAG_FRAGMENT} ${_BUILD_LABEL_FRAGMENT} ${PROJECT_ROOT} ; \

podman-push:
	for tag in ${IMAGE_TAGS}; do \
		podman push ${PODMAN_PUSH_ARGS} ${IMAGE_REPOSITORY}/${IMAGE_NAME}:$$tag ; \
	done

