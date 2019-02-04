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

install-deps: install-deps-base protoc ## Remove vendor directory, rebuild dependencies

docker-build-controller:  ## Build controller image
	(cd controller && DOCKER_IMG_NAME="controller" make docker-build)

docker-push-controller:  ## Push controller docker image to a docker hub
	(cd controller && DOCKER_IMG_NAME="controller" make docker-push)

docker-build: docker-build-base docker-build-controller        ## Install dependencies if vendor folder is missing, build go code, build docker images (includes controller).

docker-push: docker-push-base docker-push-controller           ## Push docker image to a docker hub
	(cd controller && make docker-push)

clean: clean-base                      ## clean all build artifacts
	if [ -d ./cmd/lcm/bin ]; then rm -r ./cmd/lcm/bin; fi

.PHONY: all clean doctor usage showvars test-unit
