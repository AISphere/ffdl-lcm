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

DOCKER_IMG_NAME = controller

# Because we're down a level, have to do this to get the correct AISPHERE_DIR
THIS_DIR = $(shell pwd)
PARENT_DIR ?= $(shell dirname "$(THIS_DIR)")
AISPHERE_DIR ?= $(shell dirname "$(PARENT_DIR)")

include ../../ffdl-commons/ffdl-commons.mk

STATE_MACHINE_PDF=docs/state-machine.pdf

docs:
	cat docs/state-machine.dot |docker run -i --entrypoint /usr/bin/dot andrewshawcare/graphviz -T pdf > ${STATE_MACHINE_PDF}

docker-build: docker-build-only        ## Build go code, build docker image.

docker-push: docker-push-base          ## Push docker image to a docker hub

clean:
	rm -rf build

.PHONY: build push docs clean

