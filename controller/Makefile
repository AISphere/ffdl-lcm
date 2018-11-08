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

include ../../ffdl-commons/ffdl-commons.mk

STATE_MACHINE_PDF=docs/state-machine.pdf

docs:
	cat docs/state-machine.dot |docker run -i --entrypoint /usr/bin/dot andrewshawcare/graphviz -T pdf > ${STATE_MACHINE_PDF}

docker-build: docker-build-base

clean:
	rm -rf build

.PHONY: build push docs clean

