#! vim noexpandtab
#
# Usage:
#
#   make agent-wrapper  # Makes the agent RPM, DEB and wrapper RPM
#
#   make clean          # Cleans everything
#

############################################################
# common

# For some reason, on Ubuntu 18.04, Gnu Make defaults to /bin/sh as the shell,
# breaking a bunch of things. Override it here.
SHELL := /bin/bash

############################
include version.rc
export $(shell sed 's/=.*//' version.rc)

BUILD_NUMBER ?= 0
GITHASH=$(shell git rev-parse --short HEAD)
ROOT_DIR=$(shell pwd)
BUILD_DIR=$(shell pwd)/build

# Whenever we add support for current k8s version to a DU release,
# add it to the following variable
DU_RELEASES ?= platform9-v3.0

WGET_CMD := wget --progress=dot:giga
DETECTED_OS := $(shell uname -s)
svcuser = pf9
svcgroup = pf9group

default: ;

all: agent-wrapper

clean: nodelet-clean agent-clean \
       kubernetes-test-clean \
       easyrsa-clean \
       node-clean bouncer-clean build-dir-clean

$(BUILD_DIR):
	mkdir -p $@

# Neat make target to print any defined var in the Makefile. Example: make print-SRCROOT
print-%  : ; @echo $* = $($*)

.PHONY: list
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | xargs

############################################################
# nodejs - dependency for server
.PHONY: node-linux node-clean gke-tests-clean clean image dist-container push
NODEJS_DOWNLOAD_SCRIPT = $(shell pwd)/support/download-nodejs-carbon-latest.sh
NODEJS_LINUX_DIR := $(BUILD_DIR)/node-linux
NODEJS_OSX_DIR := $(BUILD_DIR)/node-osx

ifeq ($(DETECTED_OS),Darwin)
	NPM := $(NODEJS_OSX_DIR)/bin/npm
	NODEJS_PLATFORM := $(NODEJS_OSX_DIR)
else
	NPM := $(NODEJS_LINUX_DIR)/bin/npm
	NODEJS_PLATFORM := $(NODEJS_LINUX_DIR)
endif

NODEJS := $(NODEJS_PLATFORM)/bin/node
NPM_LATEST := $(NODEJS) $(BUILD_DIR)/node_modules/.bin/npm
export PATH := $(PATH):$(NODEJS_PLATFORM)/bin

# example usage: $(call download_nodejs,"linux","/path/to/node")
define download_nodejs
tar xz --strip-components=1 -C $2 --file $(shell ${NODEJS_DOWNLOAD_SCRIPT} $1 $(BUILD_DIR))
endef

$(NODEJS_LINUX_DIR): | $(BUILD_DIR)
	mkdir -p $@
	$(call download_nodejs,"linux",$@)

$(NODEJS_OSX_DIR): | $(BUILD_DIR)
	mkdir -p $@
	$(call download_nodejs,"osx",$@)

$(NPM): | $(NODEJS_PLATFORM)
	# install latest npm
	pushd $(BUILD_DIR) && \
	$(NODEJS) $(NPM) install npm@6.14.11 && \
	popd

node-linux: | $(NODEJS_LINUX_DIR)

build-dir-clean:
	echo "Cleaning up build dir.."
	rm -rf $(BUILD_DIR)

cache-clean:
	echo "Cleaning up cache directories.."
	rm -rf /home/pf9/.kube/cache
	rm -rf /home/pf9/.kube/http-cache

node-clean:
	-@rm -rf $(NODEJS_LINUX_DIR) $(NODEJS_OSX_DIR) $(BUILD_DIR)/node*{linux,darwin}*.tar.gz 2>/dev/null || true

############################################################
# try_all_mirrors
# try to download from each mirror in the list, first
# using yum-cache http proxy, then directly.
#
# example usage: $(call try_all_mirrors,"http://foo.bar/baz http://bar.baz/foo")
#
define try_all_mirrors
declare -a mirrors=($(1)); \
for mirror in "$${mirrors[@]}"; do \
	if ${WGET_CMD} -e use_proxy=yes -e http_proxy=http://yum-cache.platform9.sys:3142 $$mirror; then break; fi; \
	if ${WGET_CMD} $$mirror; then break; fi; \
done
endef

############################################################
# Kubernetes

# For testing in isolation, you can mock parameters out like so:
# WGET_CMD="wget" QBERT_VERSION=0.1 CONF_SRC_DIR=/vagrant/pf9-kube/conf/ BUILD_DIR=./

WGET_CMD ?= "wget"

# Despotify relies on the kubernetes version to decide which kubectl it needs
# to use. Please remember to also update the version in:
# pf9-qbert/despotify/Dockerfile
KUBERNETES_GOOGLE_BASEURL := http://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_VERSION}
KUBERNETES_GITHUB_RAW_BASEURL := https://raw.githubusercontent.com/kubernetes/kubernetes/${KUBERNETES_VERSION}
KUBERNETES_GUESTBOOK_RAW_BASEURL := https://raw.githubusercontent.com/kubernetes/examples/master/guestbook/
KUBERNETES_BASEDIR := $(BUILD_DIR)/kubernetes
KUBERNETES_DIR := $(KUBERNETES_BASEDIR)/$(KUBERNETES_VERSION)
KUSTOMIZE_BASEURL := https://github.com/kubernetes-sigs/kustomize/releases/download/
KUBECTL_BIN := $(KUBERNETES_DIR)/kubectl
KUBELET_BIN := $(KUBERNETES_DIR)/kubelet
KUSTOMIZE_BIN := $(KUBERNETES_DIR)/kustomize
KUBE_GUESTBOOK_DIR := $(KUBERNETES_DIR)/guestbook
KUBERNETES_EXECUTABLES ?= /opt/pf9/pf9-kube/

############################
# check-env
# Since this build requires a few variables to be sent as inputs, we check them here.
# Not comprehensive add to this over time.
check-env:
	$(shell ./check_env.sh)

$(KUBERNETES_BASEDIR):
	echo "make kubernetes base dir: $(KUBERNETES_BASEDIR)"
	mkdir -p $@

$(KUBERNETES_DIR): | $(KUBERNETES_BASEDIR)
	echo "make kubernetes dir: $(KUBERNETES_DIR)"
	mkdir -p $@


$(KUBECTL_BIN): | $(KUBERNETES_DIR)
	echo "make kubectl binary: $(KUBECTL_BIN)"
	cd $(KUBERNETES_DIR) && ${WGET_CMD} $(KUBERNETES_GOOGLE_BASEURL)/bin/linux/amd64/kubectl
	ls -altrh $(KUBERNETES_DIR)
	chmod u=rwx,og=rx $@

$(KUBELET_BIN): | $(KUBERNETES_DIR)
	echo "make kubeLET binary: $(KUBELET_BIN)"
	cd $(KUBERNETES_DIR) && ${WGET_CMD} $(KUBERNETES_GOOGLE_BASEURL)/bin/linux/amd64/kubelet
	ls -altrh $(KUBERNETES_DIR)
	chmod u=rwx,og=rx $@

$(KUSTOMIZE_BIN): | $(KUBERNETES_DIR)
	echo "make kustomize binary: $(KUSTOMIZE_BIN)"
	cd $(KUBERNETES_DIR) && \
	${WGET_CMD} $(KUSTOMIZE_BASEURL)v$(KUSTOMIZE_VERSION)/kustomize_$(KUSTOMIZE_VERSION)_linux_amd64 && \
	mv -f kustomize_$(KUSTOMIZE_VERSION)_linux_amd64 kustomize
	ls -altrh $(KUBERNETES_DIR)
	chmod u=rwx,og=rx $@

$(KUBE_GUESTBOOK_DIR): | $(KUBERNETES_DIR)
	echo "make KUBE_GUESTBOOK_DIR $(KUBE_GUESTBOOK_DIR)"
	$(eval $@_TEMPDIR := $(shell mktemp -d))
	echo $@_TEMPDIR
	cd $($@_TEMPDIR) && \
	${WGET_CMD} $(KUBERNETES_GUESTBOOK_RAW_BASEURL)/frontend-deployment.yaml && \
	${WGET_CMD} $(KUBERNETES_GUESTBOOK_RAW_BASEURL)/frontend-service.yaml && \
	${WGET_CMD} $(KUBERNETES_GUESTBOOK_RAW_BASEURL)/redis-master-deployment.yaml && \
	${WGET_CMD} $(KUBERNETES_GUESTBOOK_RAW_BASEURL)/redis-master-service.yaml && \
	${WGET_CMD} $(KUBERNETES_GUESTBOOK_RAW_BASEURL)/redis-replica-deployment.yaml && \
	${WGET_CMD} $(KUBERNETES_GUESTBOOK_RAW_BASEURL)/redis-replica-service.yaml && \
	mv -f $($@_TEMPDIR) $@
	chmod -R u=rwX,og=rX $@

.PHONY: kubernetes kubernetes-clean kubernetes-clean-all

kubernetes: $(KUBECTL_BIN) $(KUBELET_BIN) $(KUSTOMIZE_BIN) $(KUBE_GUESTBOOK_DIR)

kubernetes-clean:
	echo "make kubernetes-clean KUBERNETES_DIR=$(KUBERNETES_DIR)"
	rm -rf $(KUBERNETES_DIR)

kubernetes-clean-all:
	echo "make kubernetes-clean KUBERNETES_BASE_DIR=$(KUBERNETES_BASEDIR)"
	rm -rf $(KUBERNETES_BASEDIR)

############################
# agent

AGENT_SRC_DIR := $(ROOT_DIR)
AGENT_TEST_DIR := $(AGENT_SRC_DIR)/test
AGENT_TEST_NODE_MODULES_DIR := $(AGENT_TEST_DIR)/node_modules
AGENT_BUILD_DIR := $(BUILD_DIR)/pf9-kube
RPMBUILD_DIR := $(AGENT_BUILD_DIR)/rpmbuild
AGENT_VERSION ?= $(KUBE_VERSION)
PF9_KUBE_VERSION := $(AGENT_VERSION)-pmk.$(BUILD_NUMBER)
PF9_KUBE_VERSION_WITH_GITHASH := $(PF9_KUBE_VERSION).$(GITHASH)
PF9_KUBE_RPM_BUILD_DIR := $(AGENT_BUILD_DIR)/rpmbuild/RPMS/x86_64
PF9_KUBE_RPM_FILE := $(PF9_KUBE_RPM_BUILD_DIR)/pf9-kube-$(PF9_KUBE_VERSION).x86_64.rpm
PF9_KUBE_DEB_FILE := $(AGENT_BUILD_DIR)/pf9-kube-$(PF9_KUBE_VERSION).x86_64.deb
PF9_KUBE_SRCDIR := $(AGENT_BUILD_DIR)/pf9-kube-src
PF9_KUBE_WRAPPER_STAGE:= $(AGENT_BUILD_DIR)/pf9-kube-wrapper-stage
PF9_KUBE_WRAPPER := $(AGENT_BUILD_DIR)/RPMS/x86_64/pf9-kube-wrapper-$(PF9_KUBE_VERSION_WITH_GITHASH).x86_64.rpm

SUPPORTED_ROLES_BUCKET = supportedroleversions
PACKAGE_BUCKET = package-repo.platform9.com
S3_URL_ROOT = s3://$(PACKAGE_BUCKET)/host-packages
PUB_URL_ROOT = https://s3-us-west-1.amazonaws.com/$(PACKAGE_BUCKET)/host-packages
INTERNAL_URL_ROOT = http://localhost:9080/private
ARTIFACTS_DIR = $(BUILD_DIR)/artifacts
PF9_KUBE_TARBALL = $(ARTIFACTS_DIR)/pf9-kube.tar.gz

############################
# EasyRSA
EASYRSA_GOOGLE_URL := https://storage.googleapis.com/kubernetes-release/easy-rsa/easy-rsa.tar.gz
EASYRSA_MIRRORS = $(EASYRSA_GOOGLE_URL)
EASYRSA_DIR := $(BUILD_DIR)/easyrsa
EASYRSA_TARBALL := $(EASYRSA_DIR)/easy-rsa.tar.gz

$(EASYRSA_DIR): | $(BUILD_DIR)
	echo "make EASYRSA_RID: $(EASYRSA_DIR)"
	mkdir -p $@

$(EASYRSA_TARBALL): | $(EASYRSA_DIR)
	echo "make EASYRSA_TARBALL $(EASYRSA_TARBALL)"
	cd $(EASYRSA_DIR) && $(call try_all_mirrors,$(EASYRSA_MIRRORS))

easyrsa: $(EASYRSA_TARBALL)

easyrsa-clean:
	rm -rf $(EASYRSA_DIR)

############################
# Agent

$(ARTIFACTS_DIR):
	mkdir -p $@

$(AGENT_BUILD_DIR):
	echo "make AGENT_BUILD_DIR $(AGENT_BUILD_DIR)"
	mkdir -p $@

$(AGENT_TEST_NODE_MODULES_DIR): $(NPM)
	echo "make AGENT_TEST_NODE_MODULES_DIR $(AGENT_TEST_NODE_MODULES_DIR)"
	cd $(AGENT_TEST_DIR) && $(NPM_LATEST) ci

$(PF9_KUBE_SRCDIR):
	echo "make PF9_KUBE_SRCDIR $(PF9_KUBE_SRCDIR)"
	rm -fr $@
	mkdir -p $@

PF9_KUBE_DEB_FILE_WITH_VERSION := $(AGENT_BUILD_DIR)/pf9-kube-$(PF9_KUBE_VERSION).x86_64.deb
COMMON_SRC_ROOT := $(PF9_KUBE_SRCDIR)/common
RPM_SRC_ROOT := $(PF9_KUBE_SRCDIR)/rpmsrc
DEB_SRC_ROOT := $(PF9_KUBE_SRCDIR)/debsrc


############################
# Kubernetes CNI and Networking

# CNI
.PHONY: cni-plugins
ARCH := linux-amd64

# To pull new cni plugin binaries, update the CNI_PLUGINS_VERSION tag to the reqired version
CNI_PLUGINS_BASE_DIR := $(BUILD_DIR)/cni-plugins
CNI_PLUGINS_DIR := $(CNI_PLUGINS_BASE_DIR)/$(CNI_PLUGINS_VERSION)
CNI_PLUGINS_FILE := cni-plugins-${ARCH}-${CNI_PLUGINS_VERSION}.tgz
CNI_PLUGINS_URL := https://github.com/containernetworking/plugins/releases/download/${CNI_PLUGINS_VERSION}/${CNI_PLUGINS_FILE}

$(CNI_PLUGINS_BASE_DIR):
	echo "make CNI_PLUGINS_BASE_DIR: $(CNI_PLUGINS_BASE_DIR)"
	mkdir -p $@

$(CNI_PLUGINS_DIR): | $(CNI_PLUGINS_BASE_DIR)
	echo "make CNI_PLUGINS_DIR: $(CNI_PLUGINS_DIR)"
	mkdir -p $@
	cd $@ && \
	${WGET_CMD} ${CNI_PLUGINS_URL}  -qO- | tar -zx

cni-plugins: $(CNI_PLUGINS_DIR)

# CONTAINERD CLI

# NERDCTL install
.PHONY: nerdctl
NERDCTL_CLI := "nerdctl"
NERDCTL_CLI_VERSION := "0.10.0"
NERDCTL_URL := https://github.com/containerd/nerdctl/releases/download/v${NERDCTL_CLI_VERSION}/nerdctl-${NERDCTL_CLI_VERSION}-linux-amd64.tar.gz
NERDCTL_DIR := "nerdctl"

nerdctl:
	echo "Downloading and installing nerdctl"
	cd ${KUBERNETES_DIR} && \
	mkdir -p ${NERDCTL_DIR} && \
    curl --output ${NERDCTL_DIR}/nerdctl-${NERDCTL_CLI_VERSION}-linux-amd64.tar.gz -L ${NERDCTL_URL} && \
    tar -C ${NERDCTL_DIR} -xvf ${NERDCTL_DIR}/nerdctl-${NERDCTL_CLI_VERSION}-linux-amd64.tar.gz


# CRICTL install
.PHONY: crictl
CRICTL_CLI := "crictl"
CRICTL_CLI_VERSION := "v1.22.0"
CRICTL_URL := https://github.com/kubernetes-sigs/cri-tools/releases/download/${CRICTL_CLI_VERSION}/crictl-${CRICTL_CLI_VERSION}-linux-amd64.tar.gz
CRICTL_DIR := "crictl"

crictl:
	echo "Downloading and installing crictl"
	cd ${KUBERNETES_DIR} && \
	mkdir -p ${CRICTL_DIR} && \
    curl --output ${CRICTL_DIR}/crictl-${CRICTL_CLI_VERSION}-linux-amd64.tar.gz -L ${CRICTL_URL} && \
    tar -C ${CRICTL_DIR} -xvf ${CRICTL_DIR}/crictl-${CRICTL_CLI_VERSION}-linux-amd64.tar.gz


###########################################################

.PHONY: calicoctl
CALICOCTL_URL := https://github.com/projectcalico/calicoctl/releases/download/${CALICOCTL_VERSION}/calicoctl

calicoctl:
	echo "Downloading and installing calicoctl"
	cd ${KUBERNETES_DIR} && \
	curl -O -L ${CALICOCTL_URL} && \
	chmod u=rwx,og=rx calicoctl

# Auth Bootstrap test
AUTHBS_SRC_DIR := $(ROOT_DIR)/authbs
AUTHBS_TEST_DIR := $(BUILD_DIR)/authbs-test
AUTHBS_TEST_SCRIPT := $(AUTHBS_TEST_DIR)/e2e-test.sh
VAULT_TEST_SCRIPT := $(AUTHBS_TEST_DIR)/vault-e2e.sh

$(AUTHBS_TEST_SCRIPT): $(AUTHBS_TEST_DIR)
	echo "RUNNING AUTHBS_TEST_SCRIPT $(AUTHBS_TEST_SCRIPT)"
	cp -a $(AUTHBS_SRC_DIR)/test/e2e-test.sh $(AUTHBS_TEST_DIR)

$(VAULT_TEST_SCRIPT): $(AUTHBS_TEST_DIR)
	echo "RUNNING VAULT_TEST_SCRIPT $(VAULT_TEST_SCRIPT)"
	cp -a $(AUTHBS_SRC_DIR)/test/vault-e2e.sh $(AUTHBS_TEST_DIR)

authbs-test: $(AUTHBS_TEST_SCRIPT)
	$(AUTHBS_TEST_SCRIPT) v1

authbs-test-v2: $(AUTHBS_TEST_SCRIPT)
	$(AUTHBS_TEST_SCRIPT) v2

authbs-test-v3: $(AUTHBS_TEST_SCRIPT)
	$(AUTHBS_TEST_SCRIPT) v3

authbs-test-vault: $(VAULT_TEST_SCRIPT)
	$(VAULT_TEST_SCRIPT)

$(AUTHBS_TEST_DIR): $(AUTHBS_SRC_DIR) $(CAPROXY) easyrsa
	mkdir -p $@
	cp -a $(AUTHBS_SRC_DIR)/reqsig $@
	cp -a $(AUTHBS_SRC_DIR)/requester $@
	cp $(EASYRSA_TARBALL) $@/requester
	cp -a $(AUTHBS_SRC_DIR)/signd $@
	cp -a $(AUTHBS_SRC_DIR)/signer $@
	cp -a $(AUTHBS_SRC_DIR)/svckeymgr $@
	cp $(EASYRSA_TARBALL) $@/signer
	cp -a $(CAPROXY) $@

authbs-test-clean:
	rm -rf $(AUTHBS_TEST_DIR)

############################################################
# CA Proxy build
CAPROXY_SRC_DIR := $(AUTHBS_SRC_DIR)/caproxy
CAPROXY_BUILD_DIR := $(BUILD_DIR)/caproxy
CAPROXY := $(CAPROXY_BUILD_DIR)/bin/caproxy

caproxy: $(CAPROXY)

export GOPATH=$(CAPROXY_BUILD_DIR)

$(CAPROXY): $(CAPROXY_SRC_DIR)/src/caproxy/*.go
	mkdir -p $(CAPROXY_BUILD_DIR)/bin
	cd $(CAPROXY_SRC_DIR)/src/caproxy && go get -d && go build -o $(CAPROXY)

caproxy-test: $(CAPROXY)
	cd $(CAPROXY_SRC_DIR)/src/caproxy && go test

caproxy-clean:
	rm -rf $(CAPROXY_BUILD_DIR)/root \
		$(CAPROXY_BUILD_DIR)/src \
		$(CAPROXY_BUILD_DIR)/bin

caproxy-clean-pkg:
	rm -rf $(CAPROXY_BUILD_DIR)/pkg

############################
# kubectl container image
.PHONY: kubectl-image kubectl-image-clean kubectl-image-upgrade
KUBECTL_IMAGE_SRC_DIR := $(ROOT_DIR)/pf9-kubectl-image
KUBECTL_IMAGE_BUILD_DIR := $(BUILD_DIR)/kubectl-image
KUBECTL_IMAGE := $(KUBECTL_IMAGE_BUILD_DIR)/pf9-kubectl.tar
KUBECTL_BIN := $(KUBERNETES_DIR)/kubectl

HELM_VERSION := v2.16.5
HELM_PLATFORM := linux-amd64
HELM_TARBALL := helm-$(HELM_VERSION)-$(HELM_PLATFORM).tar.gz
HELM_DOWNLOAD_URL := https://get.helm.sh/$(HELM_TARBALL)

kubectl-image: $(KUBECTL_IMAGE)

$(KUBECTL_IMAGE): $(KUBECTL_BIN) | $(KUBECTL_IMAGE_BUILD_DIR)
	echo "make KUBECTL image $(KUBECTL_IMAGE)"
	tar -P --xform='s,$(KUBECTL_IMAGE_SRC_DIR)/tree/,,' -czf \
		$(KUBECTL_IMAGE_BUILD_DIR)/pf9-kubectl-files.tar.gz \
		$(KUBECTL_IMAGE_SRC_DIR)/tree/*
	cp $(KUBECTL_IMAGE_SRC_DIR)/{Dockerfile,pf9-busybox.tar.gz} \
		$(KUBECTL_IMAGE_BUILD_DIR)
	cp $(KUBECTL_BIN) $(KUBECTL_IMAGE_BUILD_DIR)
	$(WGET_CMD) -P $(KUBECTL_IMAGE_BUILD_DIR) $(HELM_DOWNLOAD_URL)
	tar xf $(KUBECTL_IMAGE_BUILD_DIR)/$(HELM_TARBALL) -C $(KUBECTL_IMAGE_BUILD_DIR)
	cp $(KUBECTL_IMAGE_BUILD_DIR)/linux-amd64/helm $(KUBECTL_IMAGE_BUILD_DIR)
	docker build -t pf9-kubectl:latest $(KUBECTL_IMAGE_BUILD_DIR)
	docker save -o $(KUBECTL_IMAGE) pf9-kubectl
	docker rmi -f pf9-kubectl:latest

$(KUBECTL_IMAGE_BUILD_DIR):
	echo "make KUBECTL_IMAGE_BUILD_DIR: $(KUBECTL_IMAGE_BUILD_DIR)"
	mkdir -p $@

kubectl-image-clean:
	echo "make kubectl-image-clean: $(KUBECTL_IMAGE_BUILD_DIR)"
	docker rmi pf9-kubectl || true
	rm -rf $(KUBECTL_IMAGE_BUILD_DIR)

kubectl-image-upgrade:
	echo "make kubectl-image-upgrade for DU:  ssh=$(SSH_OPTS) kubectl=$(KUBECTL_IMAGE) du=$(DU):, "
	if [ -z $(DU) ]; then echo -e "Define DU, e.g. host or user@host:port"; exit 1; fi

	make kubectl-image-clean
	make kubectl-image

	scp $(SSH_OPTS) $(KUBECTL_IMAGE) $(DU):~/
	ssh $(SSH_OPTS) $(DU) sudo docker rmi pf9-kubectl || true
	ssh $(SSH_OPTS) $(DU) sudo docker load -i '~/pf9-kubectl.tar'

.PHONY: virtctl
VIRTCTL_URL := https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/virtctl-${KUBEVIRT_VERSION}-linux-amd64

virtctl:
	echo "Downloading and installing virtctl"
	cd ${KUBERNETES_DIR} && \
	${WGET_CMD} -O virtctl -L ${VIRTCTL_URL} && \
	chmod u=rwx,og=rx virtctl

# If this fails, the build still can continue, so we can build these RPMS outside Qbert.
-include $(ROOT_DIR)/bouncer/Makefile
-include $(ROOT_DIR)/nodelet/Makefile
-include $(ROOT_DIR)/ip_type/Makefile
-include $(ROOT_DIR)/addr_conv/Makefile

jq: 
	echo "Downloading jq"
	cd ${KUBERNETES_DIR} && \
	${WGET_CMD} -O jq -L https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64 && \
	chmod u=rwx,og=rx jq

$(COMMON_SRC_ROOT): easyrsa $(AUTHBS_SRC_DIR) bouncer-docker-image kubernetes nodelet cni-plugins nerdctl crictl calicoctl etcdctl etcd_raft_checker pf9kube-addr-conv pf9kube-ip_type virtctl jq
	echo "make COMMON_SRC_ROOT $(COMMON_SRC_ROOT)"
	echo "COMMON_SRC_ROOT is $(COMMON_SRC_ROOT)" # i.e. /vagrant/build/pf9-kube/pf9-kube-src/common
	echo "AGENT_SRC_DIR is $(AGENT_SRC_DIR)" # cp -a /vagrant/agent/root/* /vagrant/build/pf9-kube/pf9-kube-src/common/
	mkdir -p $(COMMON_SRC_ROOT)
	cp -a $(AGENT_SRC_DIR)/root/* $(COMMON_SRC_ROOT)/
	sed -i s/__KUBERNETES_VERSION__/$(KUBERNETES_VERSION)/ $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/defaults.env
	sed -i s/__FLANNEL_VERSION__/$(FLANNEL_VERSION)/ $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/defaults.env
	mkdir -p $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/conf/addons/pf9-sentry
	cp -a $(AGENT_SRC_DIR)/pf9-addons/pf9-sentry/tooling/manifests/pf9-sentry/pf9-sentry*.yaml $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/conf/addons/pf9-sentry
	mkdir -p $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/conf/addons/pf9-addon-operator
	cp -a $(AGENT_SRC_DIR)/pf9-addons/pf9-addon-operator/tooling/manifests/pf9-addon-operator/pf9-addon-operator*.yaml $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/conf/addons/pf9-addon-operator
	mkdir -p $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/
	cp -a $(KUBECTL_BIN) $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/
	cp -a $(KUBELET_BIN) $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/
	cp -a $(KUSTOMIZE_BIN) $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/
	cp -a ${KUBERNETES_DIR}/virtctl $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/
	mkdir -p $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/kubernetes/cluster
	mkdir -p $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/reqsig
	echo "-a 1"
	cp -a $(AUTHBS_SRC_DIR)/reqsig/* $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/reqsig
	mkdir -p $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/requester
	echo "requester 1 -> $(AUTHBS_SRC_DIR)/requester/* -> $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/requester "
	cp -a $(AUTHBS_SRC_DIR)/requester/* $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/requester
	echo "requester 2 $(EASYRSA_TARBALL) -> $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/requester"
	if "$(EASYRSA_TARBALL)" == "" ; then echo "missing ! EASYRSA_TARBALL value" ; exit 1 ; fi
	cp -a $(EASYRSA_TARBALL) $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/requester
	mkdir -p $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/images
	cp -a $(BOUNCER_IMAGE_TARBALL) $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/images/
	mkdir -p $(COMMON_SRC_ROOT)/opt/cni/bin
	cp -a $(CNI_PLUGINS_DIR)/* $(COMMON_SRC_ROOT)/opt/cni/bin/
	mkdir -p $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/conf/networkapps
	mv -f $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/conf/networkapps/calico.yaml $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/conf/networkapps/calico-${KUBERNETES_VERSION}.yaml
	mv -f $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/conf/networkapps/canal.yaml $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/conf/networkapps/canal-${KUBERNETES_VERSION}.yaml
	mv -f $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/conf/networkapps/weave.yaml $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/conf/networkapps/weave-${KUBERNETES_VERSION}.yaml
	mkdir -p $(COMMON_SRC_ROOT)/opt/pf9/nodelet/
	mv -f $(NODELET) $(COMMON_SRC_ROOT)/opt/pf9/nodelet/
	cp -ar $(NODELET_SRC_DIR)/tooling/tree/* $(COMMON_SRC_ROOT)/
	cp -a ${KUBERNETES_DIR}/${NERDCTL_DIR}/nerdctl $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}bin/
	cp -a ${KUBERNETES_DIR}/${CRICTL_DIR}/crictl $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}bin/
	cp -a ${KUBERNETES_DIR}/calicoctl $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/
	cp -a ${ETCD_TMP_DIR}/etcdctl $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/
	cp -a ${AGENT_BUILD_DIR}/etcd_raft_checker $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/
	cp -a ${AGENT_BUILD_DIR}/addr_conv $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/bin/
	cp -a ${AGENT_BUILD_DIR}/ip_type $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}/
	cp -a ${KUBERNETES_DIR}/jq $(COMMON_SRC_ROOT)${KUBERNETES_EXECUTABLES}bin/

$(RPM_SRC_ROOT): | $(COMMON_SRC_ROOT)
	echo "make RPM_SRC_ROOT: $(RPM_SRC_ROOT)"
	cp -a $(COMMON_SRC_ROOT) $(RPM_SRC_ROOT)

$(DEB_SRC_ROOT): | $(COMMON_SRC_ROOT)
	cp -a $(COMMON_SRC_ROOT) $(DEB_SRC_ROOT)

debug:
	echo "qbert src dir $(QBERT_SRC_DIR)"
	echo "qbert src dir ++ agent =   $(QBERT_SRC_DIR) ++ $(AGENT_BUILD_DIR)"
	echo "kube rpm $(PF9_KUBE_RPM_FILE)"

$(PF9_KUBE_RPM_FILE): | $(RPM_SRC_ROOT)
	echo "make PF9_KUBE_RPM_FILE $(PF9_KUBE_RPM_FILE) "
	rpmbuild -bb \
	    --define "_topdir $(RPMBUILD_DIR)"  \
	    --define "_src_dir $(RPM_SRC_ROOT)"  \
	    --define "_version $(AGENT_VERSION)" \
	    --define "_release pmk.$(BUILD_NUMBER)" \
	    --define "_githash $(GITHASH)" $(AGENT_SRC_DIR)/pf9-kube.spec
	./sign_packages.sh $(PF9_KUBE_RPM_FILE)
	md5sum $(PF9_KUBE_RPM_FILE) | cut -d' ' -f 1  > $(PF9_KUBE_RPM_FILE).md5

agent-rpm: check-env $(PF9_KUBE_RPM_FILE)
	echo "make agent-rpm pf9_kube_rpm_file = $(PF9_KUBE_RPM_FILE)"

$(PF9_KUBE_DEB_FILE): $(DEB_SRC_ROOT)
	fpm -t deb -s dir -n pf9-kube \
		--description "Platform9 kubernetes deb package. Built on git hash $(GITHASH)" \
		-v $(PF9_KUBE_VERSION) --provides pf9-kube --provides pf9app \
		--license "Commercial" --architecture all --url "http://www.platform9.net" --vendor Platform9 \
		-d curl -d gzip -d net-tools -d socat -d keepalived -d cgroup-tools \
		--after-install $(AGENT_SRC_DIR)/pf9-kube-after-install.sh \
		--before-remove $(AGENT_SRC_DIR)/pf9-kube-before-remove.sh \
		--after-remove ${AGENT_SRC_DIR}/pf9-kube-after-remove.sh \
		-p $(PF9_KUBE_DEB_FILE_WITH_VERSION) \
		-C $(DEB_SRC_ROOT) .
	./sign_packages_deb.sh $(PF9_KUBE_DEB_FILE)
	md5sum $(PF9_KUBE_DEB_FILE) | cut -d' ' -f 1 > $(PF9_KUBE_DEB_FILE).md5

agent-deb: $(PF9_KUBE_DEB_FILE)

$(PF9_KUBE_WRAPPER_STAGE):
	echo "make PF9_KUBE_WRAPPER_STAGE $(PF9_KUBE_WRAPPER_STAGE)"
	mkdir -p $@

$(PF9_KUBE_WRAPPER): $(PF9_KUBE_RPM_FILE) $(PF9_KUBE_WRAPPER_STAGE) $(PF9_KUBE_DEB_FILE) | $(ARTIFACTS_DIR)
	echo "make PF9_KUBE_WRAPPER rpm:$(PF9_KUBE_RPM_FILE) wrapper:$(PF9_KUBE_WRAPPER_STAGE) deb:$(PF9_KUBE_DEB_FILE)"
	cp  $(PF9_KUBE_RPM_FILE) $(PF9_KUBE_WRAPPER_STAGE)
	cp  $(PF9_KUBE_DEB_FILE) $(PF9_KUBE_WRAPPER_STAGE)
	sed -e "s/__ROLE_VERSION__/${PF9_KUBE_VERSION}/g" \
	    -e "s/__RPM_VERSION__/${PF9_KUBE_VERSION}/g" \
	    ${AGENT_SRC_DIR}/pf9-kube-role.json.template > \
	    ${PF9_KUBE_WRAPPER_STAGE}/role.json
	sed -e "s/__KUBERNETES_VERSION__/$(KUBERNETES_VERSION)/g" \
		  -e "s/__KUSTOMIZE_VERSION__/$(KUSTOMIZE_VERSION)/g" \
		  -e "s/__COREDNS_VERSION__/$(COREDNS_VERSION)/g" \
		  -e "s/__METRICS_SERVER_VERSION__/$(METRICS_SERVER_VERSION)/g" \
		  -e "s/__METALLB_VERSION__/$(METALLB_VERSION)/g" \
		  -e "s/__DASHBOARD_VERSION__/$(DASHBOARD_VERSION)/g" \
		  -e "s/__CASAWS_VERSION__/$(CASAWS_VERSION)/g" \
		  -e "s/__CASAZURE_VERSION__/$(CASAZURE_VERSION)/g" \
		  -e "s/__FLANNEL_VERSION__/$(FLANNEL_VERSION)/g" \
		  -e "s/__CALICO_VERSION__/$(CALICOCTL_VERSION)/g" \
		  -e "s/__ETCD_VERSION__/$(ETCD_VERSION)/g" \
		  -e "s/__CNI_PLUGINS_VERSION__/$(CNI_PLUGINS_VERSION)/g" \
		  -e "s/__KUBEVIRT_ADDON_VERSION__/$(KUBEVIRT_ADDON_VERSION)/g" \
		  -e "s/__LUIGI_VERSION__/$(LUIGI_VERSION)/g" \
		  -e "s/__MONITORING_VERSION__/$(MONITORING_VERSION)/g" \
		  -e "s/__PROFILE_AGENT_VERSION__/$(PROFILE_AGENT_VERSION)/g" \
			$(AGENT_SRC_DIR)/addons.json.template > ${PF9_KUBE_WRAPPER_STAGE}/addons.json
	cp $(AGENT_SRC_DIR)/metadata.json ${PF9_KUBE_WRAPPER_STAGE}/metadata.json
	sed -e "s/__BUILDNUM__/$(BUILD_NUMBER)/g" -e "s/__GITHASH__/$(GITHASH)/g" $(AGENT_SRC_DIR)/pf9-kube-wrapper.spec > $(PF9_KUBE_WRAPPER_STAGE)/pf9-kube-wrapper.spec
	rpmbuild -bb \
	    --define "_version $(AGENT_VERSION)" \
	    --define "_src_dir $(PF9_KUBE_WRAPPER_STAGE)" \
	    --define "_topdir $(AGENT_BUILD_DIR)"  $(PF9_KUBE_WRAPPER_STAGE)/pf9-kube-wrapper.spec
	./sign_packages.sh $(PF9_KUBE_WRAPPER)
	mkdir -p $(BUILD_DIR)/artifacts && \
	cp $(PF9_KUBE_RPM_FILE) $(PF9_KUBE_DEB_FILE) \
	   $(PF9_KUBE_RPM_FILE).md5 $(PF9_KUBE_DEB_FILE).md5 \
		 ${PF9_KUBE_WRAPPER_STAGE}/metadata.json \
	   ${PF9_KUBE_WRAPPER_STAGE}/role.json \
		 ${PF9_KUBE_WRAPPER_STAGE}/addons.json $(ARTIFACTS_DIR)
	echo -n "$(PF9_KUBE_VERSION)" > $(BUILD_DIR)/artifacts/component-version.txt

$(PF9_KUBE_TARBALL): | $(ARTIFACTS_DIR) $(COMMON_SRC_ROOT)
	cd $(COMMON_SRC_ROOT) && \
	echo -n "$(PF9_KUBE_VERSION)" > version.txt && \
	tar cfz $@ * && rm -f version.txt

pf9-kube-tarball: $(PF9_KUBE_TARBALL)

agent-wrapper: $(PF9_KUBE_WRAPPER)
	echo "make agent-wrapper"

agent-du-upgrade:
	echo "make agent-du-upgrade"
	if [ -z $(DU) ]; then echo -e "Define DU, e.g. user@FQDN"; exit 1; fi
	scp $(SSH_OPTS) $(ROOT_DIR)/get_build_number.sh $(DU):/tmp/
	$(eval EXISTING_BUILD_NUMBER=$(shell ssh $(SSH_OPTS) -tt $(DU) /tmp/get_build_number.sh))

	$(eval NEW_BUILD_NUMBER=$(shell expr 1 + $(EXISTING_BUILD_NUMBER)))
	BUILD_NUMBER=$(NEW_BUILD_NUMBER) make agent-wrapper

	$(eval WRAPPER_RPM=pf9-kube-wrapper-$(KUBE_VERSION)-pmk.$(NEW_BUILD_NUMBER).$(GITHASH).x86_64.rpm)

	scp $(SSH_OPTS) $(AGENT_BUILD_DIR)/RPMS/x86_64/$(WRAPPER_RPM) $(DU):~/
	ssh $(SSH_OPTS) -tt $(DU) sudo rpm --upgrade --force $(WRAPPER_RPM)

agent-clean:
	echo "make agent-clean"
	rm -rf $(AGENT_BUILD_DIR)

agent-clean-rpm:
	echo "make agent-clean-rpm"
	rm -rf $(RPM_SRC_ROOT) $(PF9_KUBE_RPM_FILE) $(COMMON_SRC_ROOT)

agent-tests: $(AGENT_TEST_NODE_MODULES_DIR)
	echo "make agent-tests"
	cd $(AGENT_TEST_DIR) && $(NODEJS) testPortChecker.js

############################
# Kubernetes test build
.PHONY: kubernetes-test kubernetes-test-clean
KUBERNETES_TEST_SRC_DIR := ${ROOT_DIR}/e2etests/kubernetes
KUBERNETES_TEST_BUILD_DIR := $(BUILD_DIR)/kubernetes-test

$(KUBERNETES_TEST_BUILD_DIR): $(KUBECTL_BIN) $(KUBERNETES_TEST_SRC_DIR)/kubetest*.sh $(KUBERNETES_TEST_SRC_DIR)/samples $(KUBERNETES_TEST_SRC_DIR)/guestbook/* $(KUBE_GUESTBOOK_DIR) $(AGENT_SRC_DIR)/root/$(KUBERNETES_EXECUTABLES)/wait_until.sh
	echo "make kuberenetes test build dir: $(KUBERNETES_TEST_BUILD_DIR)"
	mkdir -p $@
	cp -a $(KUBECTL_BIN) $(KUBERNETES_TEST_BUILD_DIR)
	cp -a $(KUBERNETES_TEST_SRC_DIR)/kubetest*.sh $@
	cp -a $(KUBERNETES_TEST_SRC_DIR)/samples $@
	mkdir -p $@/guestbook
	mkdir -p $@/nginx
	cp -a $(KUBERNETES_TEST_SRC_DIR)/guestbook/* $@/guestbook
	cp -a $(KUBERNETES_TEST_SRC_DIR)/nginx/* $@/nginx
	cp -a $(KUBE_GUESTBOOK_DIR)/* $@/guestbook
	cp -a $(AGENT_SRC_DIR)/root/${KUBERNETES_EXECUTABLES}/wait_until.sh $@/guestbook
	cp -a $(AGENT_SRC_DIR)/root/${KUBERNETES_EXECUTABLES}/wait_until.sh $@/nginx
	touch $@

kubernetes-test: $(KUBERNETES_TEST_BUILD_DIR)
	echo "make kuberenetes-test : $(KUBERNETES_TEST_BUILD_DIR)"

kubernetes-test-clean:
	echo "make kubernetes-test-clean: removing $(KUBERNETES_TEST_BUILD_DIR)"
	rm -rf $(KUBERNETES_TEST_BUILD_DIR)

############################
# etcdctl binary
.PHONY: etcdctl etcd-clean

ETCD_DOWNLOAD_URL := https://storage.googleapis.com/etcd
ETCD_TMP_DIR := ${AGENT_BUILD_DIR}/etcd

etcdctl:
	mkdir -p ${ETCD_TMP_DIR}
	cd ${ETCD_TMP_DIR} && ${WGET_CMD} ${ETCD_DOWNLOAD_URL}/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-amd64.tar.gz
	tar xzvf ${ETCD_TMP_DIR}/etcd-${ETCD_VERSION}-linux-amd64.tar.gz -C ${ETCD_TMP_DIR} --strip-components=1

etcd-clean:
	rm -rf ${ETCD_TMP_DIR}

############################
# Raft Index Checker
.PHONY: etcd_raft_checker etcd_raft_checker_clean
ETCD_RAFT_CHECKER_SRC_DIR := $(AGENT_SRC_DIR)/etcd_raft_checker
etcd_raft_checker: $(ETCD_RAFT_CHECKER_SRC_DIR)/*.go
	echo "building etcd_raft_checker"
	cd $(ETCD_RAFT_CHECKER_SRC_DIR) && go build -o $(AGENT_BUILD_DIR)/etcd_raft_checker

etcd_raft_checker_clean:
	rm -f $(AGENT_BUILD_DIR)/etcd/etcd_raft_checker

upload-host-packages:
	component_version=`cat $(BUILD_DIR)/artifacts/component-version.txt` && \
			echo $(component_version) && \
			md5ext=".md5" && \
			for ext in rpm deb; do \
					pkg="pf9-kube-$${component_version}.x86_64.$${ext}"; \
					path=$(BUILD_DIR)/artifacts/$${pkg}; \
					s3url=$(S3_URL_ROOT)/pf9-kube/$${component_version}/$${pkg}; \
					puburl=$(PUB_URL_ROOT); \
					internalurl=$(INTERNAL_URL_ROOT); \
					aws s3 cp --acl public-read $${path} $${s3url}; \
					aws s3 cp --acl public-read "$${path}$${md5ext}" "$${s3url}$${md5ext}"; \
			done
	aws s3 cp --acl public-read $(BUILD_DIR)/artifacts/addons.json $(S3_URL_ROOT)/pf9-kube/$(PF9_KUBE_VERSION)/addons.json;
	aws s3 cp --acl public-read $(BUILD_DIR)/artifacts/metadata.json $(S3_URL_ROOT)/pf9-kube/$(PF9_KUBE_VERSION)/metadata.json;
	aws s3 cp --acl public-read $(BUILD_DIR)/artifacts/role.json $(S3_URL_ROOT)/pf9-kube/$(PF9_KUBE_VERSION)/role.json;

update-supported-version:
	mkdir -p $(BUILD_DIR)/artifacts
	touch $(BUILD_DIR)/artifacts/$(PF9_KUBE_VERSION);
	for du_release in $(DU_RELEASES); do \
		s3_roles_root="s3://$(SUPPORTED_ROLES_BUCKET)/$$du_release"; \
		echo "promoting to $$s3_roles_root"; \
		aws s3 cp --acl public-read $(BUILD_DIR)/artifacts/$(PF9_KUBE_VERSION) $${s3_roles_root}/$(PF9_KUBE_VERSION); \
	done

