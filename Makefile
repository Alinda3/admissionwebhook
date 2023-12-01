DOCKER_IMAGE_NAME=stylelabsinfra.azurecr.io/admission-webhook-nodeselector

# Tag your Docker image with the corresponding kubectl version installed within
DOCKER_IMAGE_TAG=v1.25.0

.PHONY: docker-build
docker-build:
	docker build --build-arg KUBECTL_VERSION="$(DOCKER_IMAGE_TAG)" -t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) . 

.PHONY: docker-push
docker-push:
	docker push $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) 
