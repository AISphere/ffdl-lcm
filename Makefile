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

# NOTE: should this variables section be is ffdl-commons and included in respective service makefiles?

# The ip or hostname of the Docker host.
# Note the awkward name is to avoid clashing with the DOCKER_HOST variable.
DOCKERHOST_HOST ?= localhost

ifeq ($(DOCKERHOST_HOST),localhost)
 # Check if minikube is active, otherwise leave it as 'localhost'
 MINIKUBE_IP := $(shell minikube ip 2>/dev/null)
 ifdef MINIKUBE_IP
  DOCKERHOST_HOST := $(MINIKUBE_IP)
 endif
endif

FSWATCH := $(shell which fswatch 2>/dev/null)

WHOAMI ?= $(shell whoami)

DOCKER_BX_NS ?= registry.ng.bluemix.net/dlaas_dev
DOCKER_BASE_IMG_NAME=dlaas-service-base
DOCKER_BASE_IMG_TAG=ubuntu16.04
SWAGGER_FILE=api/swagger/swagger.yml

DOCKER_IMG_NAME=lifecycle-manager-service

KUBE_CURRENT_CONTEXT ?= $(shell kubectl config current-context)

# The environment where DLaaS is deployed.
# This affects a number of other variables below.
ifeq ($(KUBE_CURRENT_CONTEXT), minikube)
 # Automatically set to local if Kubernetes context is "minikube"
 DLAAS_ENV ?= local
 DLAAS_LCM_DEPLOYMENT ?= hybrid
else
 DLAAS_ENV ?= development
 DLAAS_LCM_DEPLOYMENT ?= development
endif

# Support two different Kuberentes clusters:
# - one to deploy the DLaaS microservices
# - one to deploy the learners and parameter servers.
DLAAS_SERVICES_KUBE_CONTEXT ?= $(KUBE_CURRENT_CONTEXT)
DLAAS_LEARNER_KUBE_CONTEXT ?= $(KUBE_CURRENT_CONTEXT)

# For non-local deployments, Kubernetes namespace
ifeq ($(DLAAS_ENV), local)
 export INVENTORY ?= ansible/envs/local/minikube.ini
 DLAAS_IMAGE_PULL_POLICY ?= IfNotPresent   # needed ?
 LCM_SERVICE_CPU_REQ ?= 100m               # needed ?
 LCM_SERVICE_MEMORY_REQ ?= 64Mi            # needed ?
else
 INVENTORY ?= ansible/envs/local/hybrid.ini
 DLAAS_IMAGE_PULL_POLICY ?= Always         # needed ?
 LCM_SERVICE_CPU_REQ ?= "1"                # needed ?
 LCM_SERVICE_MEMORY_REQ ?= 512Mi           # needed ?
endif

DLAAS_SERVICES_KUBE_NAMESPACE ?= $(shell env DLAAS_KUBE_CONTEXT=$(DLAAS_SERVICES_KUBE_CONTEXT) ./bin/kubecontext.sh namespace)
DLAAS_LEARNER_KUBE_NAMESPACE ?= $(shell env DLAAS_KUBE_CONTEXT=$(DLAAS_LEARNER_KUBE_CONTEXT) ./bin/kubecontext.sh namespace)
DLAAS_LEARNER_KUBE_URL ?= $(shell env DLAAS_KUBE_CONTEXT=$(DLAAS_LEARNER_KUBE_CONTEXT) ./bin/kubecontext.sh api-server)
DLAAS_LEARNER_KUBE_CAFILE ?= $(shell env DLAAS_KUBE_CONTEXT=$(DLAAS_LEARNER_KUBE_CONTEXT) ./bin/kubecontext.sh server-certificate)
DLAAS_LEARNER_KUBE_TOKEN ?= $(shell env DLAAS_KUBE_CONTEXT=$(DLAAS_LEARNER_KUBE_CONTEXT) ./bin/kubecontext.sh user-token)
DLAAS_LEARNER_KUBE_KEYFILE ?= $(shell env DLAAS_KUBE_CONTEXT=$(DLAAS_LEARNER_KUBE_CONTEXT) ./bin/kubecontext.sh client-key)
DLAAS_LEARNER_KUBE_CERTFILE ?= $(shell env DLAAS_KUBE_CONTEXT=$(DLAAS_LEARNER_KUBE_CONTEXT) ./bin/kubecontext.sh client-certificate)
DLAAS_LEARNER_KUBE_SECRET ?= kubecontext-$(DLAAS_LEARNER_KUBE_CONTEXT)


KUBE_SERVICES_CONTEXT_ARGS = --context $(DLAAS_SERVICES_KUBE_CONTEXT) --namespace $(DLAAS_SERVICES_KUBE_NAMESPACE)
KUBE_LEARNER_CONTEXT_ARGS = --context $(DLAAS_LEARNER_KUBE_CONTEXT) --namespace $(DLAAS_LEARNER_KUBE_NAMESPACE)

# Use non-conflicting image tag, and Eureka name.
DLAAS_IMAGE_TAG ?= user-$(WHOAMI)
DLAAS_EUREKA_NAME ?= $(shell echo DLAAS-USER-$(WHOAMI) | tr '[:lower:]' '[:upper:]')

DLAAS_LEARNER_TAG?=dev_v8

# The target host for the e2e test.
#DLAAS_HOST?=localhost:30001
DLAAS_HOST?=$(shell env DLAAS_KUBE_CONTEXT=$(DLAAS_SERVICES_KUBE_CONTEXT) ./bin/kubecontext.sh restapi-url)

# The target host for the grpc cli.
DLAAS_GRPC?=$(shell env DLAAS_KUBE_CONTEXT=$(DLAAS_SERVICES_KUBE_CONTEXT) ./bin/kubecontext.sh trainer-url)

LEARNER_DEPLOYMENT_ARGS = DLAAS_LEARNER_KUBE_URL=$(DLAAS_LEARNER_KUBE_URL) \
                          DLAAS_LEARNER_KUBE_TOKEN=$(DLAAS_LEARNER_KUBE_TOKEN) \
                          DLAAS_LEARNER_KUBE_KEYFILE=$(DLAAS_LEARNER_KUBE_KEYFILE) \
                          DLAAS_LEARNER_KUBE_CERTFILE=$(DLAAS_LEARNER_KUBE_CERTFILE) \
                          DLAAS_LEARNER_KUBE_CAFILE=$(DLAAS_LEARNER_KUBE_CAFILE) \
                          DLAAS_LEARNER_KUBE_NAMESPACE=$(DLAAS_LEARNER_KUBE_NAMESPACE) \
                          DLAAS_LEARNER_KUBE_SECRET=$(DLAAS_LEARNER_KUBE_SECRET)
DEPLOYMENT_ARGS = DLAAS_ENV=$(DLAAS_ENV) $(LEARNER_DEPLOYMENT_ARGS) \
                  DLAAS_LCM_DEPLOYMENT=$(DLAAS_LCM_DEPLOYMENT) \
                  DLAAS_IMAGE_TAG=$(DLAAS_IMAGE_TAG) \
                  DLAAS_LEARNER_TAG=$(DLAAS_LEARNER_TAG) \
                  DLAAS_IMAGE_PULL_POLICY=$(DLAAS_IMAGE_PULL_POLICY) \
                  LCM_SERVICE_CPU_REQ=$(LCM_SERVICE_CPU_REQ) \
                  LCM_SERVICE_MEMORY_REQ=$(LCM_SERVICE_MEMORY_REQ) \
                  DLAAS_ETCD_ADDRESS=$(DLAAS_ETCD_ADDRESS) \
				  DLAAS_ETCD_PREFIX=$(DLAAS_ETCD_PREFIX) \
				  DLAAS_ETCD_USERNAME=$(DLAAS_ETCD_USERNAME) \
				  DLAAS_ETCD_PASSWORD=$(DLAAS_ETCD_PASSWORD) \
				  DLAAS_MOUNTCOS_GB_CACHE_PER_GPU=$(DLAAS_MOUNTCOS_GB_CACHE_PER_GPU)

BUILD_DIR=build

THIS_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

usage:              ## Show this help
	@fgrep -h " ## " $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

TRAINER_REPO ?= raw.githubusercontent.com/sboagibm/ffdl-trainer
TRAINER_VERSION ?= proto-only-depend
TRAINER_LOCATION ?= vendor/github.com/AISphere/ffdl-trainer
TRAINER_SUBDIR ?= trainer/grpc_trainer_v2
TRAINER_SUBDIR_IN ?= trainer/grpc_trainer_v2
TRAINER_FNAME ?= trainer

protoc-trainer:  ## Make the trainer protoc client, depends on `make glide` being run first
	#	rm -rf $(TRAINER_LOCATION)/$(TRAINER_SUBDIR)
	wget https://$(TRAINER_REPO)/$(TRAINER_VERSION)/$(TRAINER_SUBDIR_IN)/$(TRAINER_FNAME).proto -P $(TRAINER_LOCATION)/$(TRAINER_SUBDIR)
	cd ./$(TRAINER_LOCATION); \
	protoc -I./$(TRAINER_SUBDIR) --go_out=plugins=grpc:$(TRAINER_SUBDIR) ./$(TRAINER_SUBDIR)/$(TRAINER_FNAME).proto
	@# At the time of writing, protoc does not support custom tags, hence use a little regex to add "bson:..." tags
	@# See: https://github.com/golang/protobuf/issues/52
	cd $(TRAINER_LOCATION); \
	sed -i .bak '/.*bson:.*/! s/json:"\([^"]*\)"/json:"\1" bson:"\1"/' ./$(TRAINER_SUBDIR)/$(TRAINER_FNAME).pb.go

vet:
	go vet $(shell glide nv)

lint:               ## Run the code linter
	go list ./... | grep -v /vendor/ | grep -v /grpc_trainer_v2 | xargs -L1 golint -set_exit_status

glide:               ## Run full glide rebuild
	glide cache-clear; \
	rm -rf vendor; \
	glide install

#####################################################
## Extra helpers for the LCM, but which are deployed
## in the trainer pod.
#####################################################

docker-build-controller:
	(cd cmd/lcm/controller && IMAGE_TAG=$(DLAAS_IMAGE_TAG) make build)

docker-push-controller:
	(cd cmd/lcm/controller && IMAGE_TAG=$(DLAAS_IMAGE_TAG) make push)

docker-build-lcm-helpers: docker-build-controller

docker-push-lcm-helpers: docker-push-controller

build-x86-64-lcm:
	cd vendor/github.com/AISphere/ffdl-commons/grpc-health-checker && make install-deps build-x86-64
	(CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo -o bin/main)

build-x86-64-jobmonitor:
	(cd ../ffdl-job-monitor/ && make build-x86-64-jobmonitor)

build-x86-64: build-x86-64-lcm build-x86-64-jobmonitor docker-build-lcm-helpers

docker-build-lcm: build-x86-64
	cd vendor/github.com/AISphere/ffdl-commons/grpc-health-checker && make install-deps build-x86-64
	(docker build --label git-commit=$(shell git rev-list -1 HEAD) -t "$(DOCKER_BX_NS)/$(DOCKER_IMG_NAME):$(DLAAS_IMAGE_TAG)" .)

docker-build:       ## Build the Docker image
docker-build: docker-build-lcm build-x86-64-jobmonitor

docker-push:       ## Push the Docker images to the registry
docker-push: docker-push-lcm docker-push-jobmonitor

docker-push-lcm-only:
	docker push "$(DOCKER_BX_NS)/lifecycle-manager-service:$(DLAAS_IMAGE_TAG)"

docker-push-lcm: docker-push-lcm-only docker-push-lcm-helpers

docker-push-jobmonitor:
	docker push "$(DOCKER_BX_NS)/jobmonitor:$(DLAAS_IMAGE_TAG)"


# Define environment variables for unit and integration testing
DLAAS_MONGO_PORT ?= 27017

#these credentials should be the same as what are present in lcm-secrets
DLAAS_ETCD_ADDRESS=https://watson-dev3-dal10-10.compose.direct:15232,https://watson-dev3-dal10-9.compose.direct:15232
DLAAS_ETCD_USERNAME=root
DLAAS_ETCD_PASSWORD=RHDACXYDLMIXXPEE
DLAAS_ETCD_PREFIX=/dlaas/jobs/local_hybrid/

test-start-deps:   ## Start test dependencies
	docker run -d -p $(DLAAS_MONGO_PORT):27017 --name mongo mongo:3.0

# Stop test dependencies
test-stop-deps:
	-docker rm -f mongo

TEST_PKGS ?= $(shell go list ./... | grep -v /vendor/)

test-unit:         ## Run all unit tests (short tests)
	DLAAS_LOGLEVEL=debug DLAAS_ENV=local go test $(TEST_PKGS) -v -short

test-integration:  ## Run all integration tests (non-short tests with Integration in the name)
	DLAAS_LOGLEVEL=debug DLAAS_DNS_SERVER=disabled DLAAS_ENV=local  go test $(TEST_PKGS) -run "Integration" -v

test-lcm:
	DLAAS_LOGLEVEL=debug DLAAS_HOST=$(DLAAS_HOST) $(DEPLOYMENT_ARGS) go test github.ibm.com/deep-learning-platform/ffdl-lcm/service/lcm -v

# Runs unit and integration tests
test: test-unit test-integration


RESTAPI_SERVICE = ../dlaas-restapi-service
TRAINER_SERVICE = ../dlaas-trainer-service
TRAINING_DATA_SERVICE = ../dlaas-training-metrics-service
RATELIMITER_SERVICE = ../dlaas-ratelimiter

DLAAS_LOCAL_ARGS = DLAAS_LOGLEVEL=debug DLAAS_HOST=$(DLAAS_HOST) \
		$(DEPLOYMENT_ARGS) \
		DLAAS_ENV=local \
		DLAAS_LCM_DEPLOYMENT=local \
		DLAAS_DNS_SERVER=disabled

serve-local:
ifndef FSWATCH
	@echo "ERROR: fswatch not found. Please install it to use this target."
	@exit 1
endif
	make kill-services-local
	make run-services-local
	fswatch -r -o cmd config logger services storage util *.yml | xargs -n1 -I{}  make run-services-local || make kill-services-local

serve-local-lcm:
ifndef FSWATCH
	@echo "ERROR: fswatch not found. Please install it to use this target."
	@exit 1
endif
	make kill-local-lcm
	make run-local-lcm
	fswatch -r -o cmd/lcm config logger services storage util *.yml | xargs -n1 -I{}  make run-local-lcm || make kill-local-lcm

run-services-local: kill-services-local run-local-lcm run-local-trainer run-local-training-data run-local-restapi

run-local-restapi:
	(cd $(RESTAPI_SERVICE) && make run-local)

run-local-trainer:
	(cd $(TRAINER_SERVICE) && make run-local)

run-local-lcm:
	( go build -i -o ./bin/ffdl-lcm ./cmd/lcm/main.go && $(DLAAS_LOCAL_ARGS) DLAAS_PORT=30002 ./bin/ffdl-lcm & )

run-local-training-data:
	(cd $(TRAINING_DATA_SERVICE) && make run-local)


# echo exact environment variables used to launch local services, for debugging and the like.
showrunlocalvars: showvars
	@echo "# =========== env grep dlaas vars ============"
	@for ln in $(shell $(DLAAS_LOCAL_ARGS) env | grep DLAAS_); do \
		echo "export $$ln"; \
	done

LOCALEXECCOMMAND ?= MUST_SET_LOCALEXECCOMMAND

# exec-local is a dev special for executing something with the same environment that run-services-local uses.
exec-local:
	$(shell $(DLAAS_LOCAL_ARGS) $(LOCALEXECCOMMAND))

kill-services-local:
	(cd $(RESTAPI_SERVICE) && make kill-local)
	(cd $(TRAINER_SERVICE) && make kill-local)
	(cd $(TRAINING_DATA_SERVICE) && make kill-local)
	-killall ffdl-lcm

kill-local-lcm:
	-killall ffdl-lcm

kube-artifacts:    ## Show the state of various Kubernetes artifacts
	kubectl $(KUBE_SERVICES_CONTEXT_ARGS) get pod,configmap,svc,ing,statefulset,job,pvc,deploy,secret -o wide --show-all
	#@echo; echo
	#kubectl $(KUBE_LEARNER_CONTEXT_ARGS) get deploy,statefulset,pod,pvc -o wide --show-all

kube-destroy:
	@echo "If you're sure you want to delete the $(DLAAS_SERVICES_KUBE_NAMESPACE)" namespace, run the following command:
	@echo "  kubectl $(KUBE_SERVICES_CONTEXT_ARGS) delete namespace $(DLAAS_SERVICES_KUBE_NAMESPACE)"

# This list is the union of needs for all services in this Makefile
DEPLOY_EXTRA_VARS = --extra-vars "service_version=$(DLAAS_IMAGE_TAG)" \
		--extra-vars "DLAAS_NAMESPACE=$(DLAAS_SERVICES_KUBE_NAMESPACE)" \
		--extra-vars "DLAAS_LEARNER_KUBE_NAMESPACE=$(DLAAS_LEARNER_KUBE_NAMESPACE)" \
		--extra-vars "DLAAS_LEARNER_KUBE_URL=$(DLAAS_LEARNER_KUBE_URL)" \
		--extra-vars "dlaas_learner_tag=$(DLAAS_LEARNER_TAG)" \
		--extra-vars "eureka_name=$(DLAAS_EUREKA_NAME)"

#devstack-start: sv-setup   ## Start up the local dev stack
#	-docker login -u token -p `cat certs/bluemix-cr-ng-token` registry.ng.bluemix.net
#	-kubectl create secret docker-registry bluemix-cr-ng --docker-username token --docker-password `cat certs/bluemix-cr-ng-token` --docker-server registry.ng.bluemix.net --docker-email wps@us.ibm.com
#	ANSIBLE_HOST_KEY_CHECKING=False ANSIBLE_ROLES_PATH=$(THIS_DIR)/ansible/roles \
#		ansible-playbook -b -i $(INVENTORY) ansible/plays/ffdl-devstack-k8s.yml \
#		-c local \
#		--extra-vars "service=mongo" \
#		$(DEPLOY_EXTRA_VARS)
#	bin/copy_learner_config.sh devwat-dal13-cruiser15-dlaas $(DLAAS_SERVICES_KUBE_CONTEXT)
##	(cd $(TRAINER_SERVICE) && make devstack-start)

devstack-stop:
	-kubectl $(KUBE_SERVICES_CONTEXT_ARGS) delete service mongo --ignore-not-found=true --now
	-kubectl $(KUBE_SERVICES_CONTEXT_ARGS) delete statefulset mongo-deployment --ignore-not-found=true --now
	-kubectl $(KUBE_SERVICES_CONTEXT_ARGS) delete configmap learner-config --ignore-not-found=true --now
#	(cd $(TRAINER_SERVICE) && make devstack-stop)

devstack-restart: devstack-stop devstack-start

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

#deploy-fluentd:
#	DLAAS_KUBE_CONTEXT=$(DLAAS_LEARNER_KUBE_CONTEXT) ./bin/create-secret.sh $(DLAAS_LEARNER_KUBE_SECRET)
#	ANSIBLE_HOST_KEY_CHECKING=False ANSIBLE_ROLES_PATH=$(THIS_DIR)/ansible/roles \
#		ansible-playbook -b -i $(INVENTORY) ansible/plays/ffdl-fluentd-k8s.yml \
#		-c local \
#		--verbose \
#		--extra-vars "operation=apply" \
#		--extra-vars "DLAAS_LEARNER_KUBE_SECRET=$(DLAAS_LEARNER_KUBE_SECRET)" \
#		$(DEPLOY_EXTRA_VARS)
#
#undeploy-fluentd:
#	DLAAS_KUBE_CONTEXT=$(DLAAS_LEARNER_KUBE_CONTEXT) ./bin/create-secret.sh $(DLAAS_LEARNER_KUBE_SECRET)
#	ANSIBLE_HOST_KEY_CHECKING=False ANSIBLE_ROLES_PATH=$(THIS_DIR)/ansible/roles \
#		ansible-playbook -b -i $(INVENTORY) ansible/plays/dlaas-fluentd-k8s.yml \
#		-c local \
#		--verbose \
#		--extra-vars "operation=delete" \
#		--extra-vars "DLAAS_LEARNER_KUBE_SECRET=$(DLAAS_LEARNER_KUBE_SECRET)" \
#		$(DEPLOY_EXTRA_VARS)
#
#
#ansible-setup-ubuntu:
#	sudo apt-add-repository -y ppa:ansible/ansible
#	sudo apt-get update
#	sudo apt-get install -y ansible
#
## Will only execute tasks tagged (flag -t) 'setup' in Ansible role
#sv-setup:
#	DLAAS_KUBE_CONTEXT=$(DLAAS_LEARNER_KUBE_CONTEXT)
#	ANSIBLE_HOST_KEY_CHECKING=False ANSIBLE_ROLES_PATH=$(THIS_DIR)/ansible/roles \
#			ansible-playbook -b -i $(INVENTORY) -t setup $(THIS_DIR)/ansible/plays/dlaas-static-pvc-k8s.yml \
#			-c local \
#			--verbose \
#			--extra-vars "DLAAS_LEARNER_KUBE_SECRET=$(DLAAS_LEARNER_KUBE_SECRET)" \
#			$(DEPLOY_EXTRA_VARS)
#
## Will only execute tasks tagged (flag -t) 'delete' in Ansible role
#sv-delete:
#	DLAAS_KUBE_CONTEXT=$(DLAAS_LEARNER_KUBE_CONTEXT)
#	ANSIBLE_HOST_KEY_CHECKING=False ANSIBLE_ROLES_PATH=$(THIS_DIR)/ansible/roles \
#			ansible-playbook -b -i $(INVENTORY) -t delete $(THIS_DIR)/ansible/plays/dlaas-static-pvc-k8s.yml \
#			-c local \
#			--verbose \
#			--extra-vars "DLAAS_LEARNER_KUBE_SECRET=$(DLAAS_LEARNER_KUBE_SECRET)" \
#			$(DEPLOY_EXTRA_VARS)

clean:
	if [ -d ./cmd/lcm/bin ]; then rm -r ./cmd/lcm/bin; fi

.PHONY: all clean doctor usage showvars test-unit
