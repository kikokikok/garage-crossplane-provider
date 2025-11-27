IMAGE_PLATFORMS ?= linux/amd64
IMAGE ?= provider-garage

# Extract OS and ARCH from IMAGE_PLATFORMS (assuming single platform)
OS := $(word 1,$(subst /, ,$(IMAGE_PLATFORMS)))
ARCH := $(word 2,$(subst /, ,$(IMAGE_PLATFORMS)))

# BUILD_ARGS can be passed from parent Makefile (e.g., --load for buildx)
BUILD_ARGS ?=

img.build:
	@echo "Preparing binary for $(IMAGE_PLATFORMS)"
	mkdir -p bin/$(OS)_$(ARCH)
	cp ../../../_output/bin/$(OS)_$(ARCH)/provider bin/$(OS)_$(ARCH)/provider
	@echo "Building docker image $(IMAGE)"
	docker buildx build --platform $(IMAGE_PLATFORMS) $(BUILD_ARGS) -t $(IMAGE) .

img.publish:
	docker push $(IMAGE)

img.promote:
	docker tag $(FROM_IMAGE) $(TO_IMAGE)
	docker push $(TO_IMAGE)
