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

protoc: protoc-trainer

test-start-deps:   ## Start test dependencies
	docker run -d -p $(DLAAS_MONGO_PORT):27017 --name mongo mongo:3.0

# Stop test dependencies
test-stop-deps:
	-docker rm -f mongo

TEST_PKGS ?= $(shell go list ./... | grep -v /vendor/)

# Runs unit and integration tests
test: test-base

LOCALEXECCOMMAND ?= MUST_SET_LOCALEXECCOMMAND

# Add a route on OS X to access docker instances directly
#
route-add-osx:
ifeq ($(shell uname -s),Darwin)
	sudo route -n add -net 172.17.0.0 $(DOCKERHOST_HOST)
endif

# Function for generating a template
define render_template
	eval "echo \"$$(cat $(1))\""
endef

# Total reinstall of vendor directories in all services.
glide-reinstall-all:
	glide cache-clear
	rm -rf vendor && glide install
	(cd $(TRAINER_SERVICE) && rm -rf vendor && glide install)
	(cd $(TRAINING_DATA_SERVICE) && rm -rf vendor && glide install)
	(cd $(RESTAPI_SERVICE) && rm -rf vendor && glide install)
	(cd $(RATELIMITER_SERVICE) && rm -rf vendor && glide install)

deploy-services: deploy-trainer deploy-lcm deploy-restapi deploy-training-data deploy-ratelimiter
undeploy-services: undeploy-lcm undeploy-trainer undeploy-restapi undeploy-training-data undeploy-ratelimiter
redeploy-services: undeploy-services deploy-services
redeploy-lcm: undeploy-lcm deploy-lcm
redeploy-trainer: undeploy-trainer deploy-trainer
redeploy-restapi: undeploy-restapi deploy-restapi
redeploy-training-data: undeploy-training-data deploy-training-data
redeploy-ratelimiter: undeploy-ratelimiter deploy-ratelimiter

deploy-trainer:
	(cd $(TRAINER_SERVICE) && make deploy)

undeploy-trainer:
	(cd $(TRAINER_SERVICE) && make undeploy)

deploy-training-data:
	(cd $(TRAINING_DATA_SERVICE) && make deploy)

undeploy-training-data:
	(cd $(TRAINING_DATA_SERVICE) && make undeploy)

deploy-ratelimiter:
	(cd $(RATELIMITER_SERVICE) && make deploy)

undeploy-ratelimiter:
	(cd $(RATELIMITER_SERVICE) && make undeploy)

show-inventory-file:
	(echo $(INVENTORY))

install-deps: install-deps-base protoc

docker-build: docker-build-base
	(cd controller && make docker-build)

clean: clean-base
	if [ -d ./cmd/lcm/bin ]; then rm -r ./cmd/lcm/bin; fi

.PHONY: all clean doctor usage showvars test-unit
