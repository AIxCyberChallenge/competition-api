IMAGE_NAME := competition-api

# make skips the if check without an unconditional use of the variable outside of the if condition...
$(eval dummy := $(PROJECT_ROOT))
$(eval dummy := $(CHANGED_FILES))
ifdef PROJECT_ROOT
	SOURCES := $(shell find ${PROJECT_ROOT}/competition-api -type f) ${PROJECT_ROOT}/.github/workflows/images.yaml ${PROJECT_ROOT}/Makefile ${PROJECT_ROOT}/container.mk
	include ${PROJECT_ROOT}/container.mk
endif
