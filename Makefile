# Image URL to use all building/pushing image targets
source .settings

# Build the container image
.PHONY: build
build:
	docker build -t ${URL}${NAME}:${VERSION} -t ${URL}${NAME}:latest .

# Push the container image
.PHONY: push
push:
	aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws
	docker push ${URL}${NAME}:${VERSION}
	docker push ${URL}${NAME}:latest

# Default target
.DEFAULT_GOAL := build
