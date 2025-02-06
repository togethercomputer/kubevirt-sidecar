# Image URL to use all building/pushing image targets
include .settings

# Build the container image
.PHONY: build
build:
	docker build --platform linux/amd64 -t ${URL}${NAME}:v${VERSION} -t ${URL}${NAME}:latest .

# Push the container image
.PHONY: push
push:
	aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws
	docker push ${URL}${NAME}:v${VERSION}
	docker push ${URL}${NAME}:latest

.PHONY: build-branch
build-branch:	
	docker build --platform linux/amd64 -t ${URL}${NAME}:$(shell git rev-parse --abbrev-ref HEAD) .

# Push a tagged container image
.PHONY: push-branch
push-branch:
	aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws
	docker push ${URL}${NAME}:$(shell git rev-parse --abbrev-ref HEAD)	

# Default target
.DEFAULT_GOAL := build
