#
# Copyright 2017-2018 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

#
# Build and deploy file for the ffdl-lcm service
#

DOCKER_IMG_NAME = lifecycle-manager-service

include ../ffdl-commons/ffdl-commons.mk

protoc: protoc-trainer                 ## Build gRPC .proto files into vendor directory

install-deps: install-deps-base protoc build-health-checker-deps ## Remove vendor directory, rebuild dependencies

glide-update: glide-update-base        ## Run full glide rebuild

# === Health Checker Sub Build ===

build-health-checker-deps: clean-health-checker-builds
	mkdir build
	cp -r vendor/github.com/AISphere/ffdl-commons/grpc-health-checker build/
	cd build/grpc-health-checker && make build-x86-64
	cp -r build ./controller/build
	cp -r build ./jmbuild/build

# === Job Monitor Build ===

build-x86-64-jobmonitor:
	(cd ./jmbuild/ && rm -rf bin && CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo -o bin/main)

docker-build-jobmonitor: install-deps-if-needed
	make build-x86-64-jobmonitor
	(cd ./jmbuild/ && docker build --label git-commit=$(shell git rev-list -1 HEAD) -t "$(DOCKER_HOST_NAME)/$(DOCKER_NAMESPACE)/jobmonitor:$(DLAAS_IMAGE_TAG)" .)

docker-push-jobmonitor:
	docker push "$(DOCKER_HOST_NAME)/$(DOCKER_NAMESPACE)/jobmonitor:$(DLAAS_IMAGE_TAG)"

# === Controller Build ===

docker-build-controller:  ## Build controller image
	(cd controller && DOCKER_IMG_NAME="controller" make docker-build)

docker-push-controller:  ## Push controller docker image to a docker hub
	(cd controller && DOCKER_IMG_NAME="controller" make docker-push)

# === Service Build ===

docker-build: docker-build-base docker-build-controller docker-build-jobmonitor     ## Install dependencies if vendor folder is missing, build go code, build docker images (includes controller).

docker-push: docker-push-base docker-push-controller docker-push-jobmonitor         ## Push docker image to a docker hub

clean-health-checker-builds:
	rm -rf ./build
	rm -rf ./controller/build
	rm -rf ./jmbuild/build

clean-helper-bins:
	rm -rf ./bin
	rm -r ./jmbuild/bin

clean: clean-base clean-health-checker-builds clean-helper-bin                  ## clean all build artifacts

.PHONY: all clean doctor usage showvars test-unit
