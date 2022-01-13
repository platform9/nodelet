CGO_ENABLED?=0
GOOS?=linux
GOARCH?=amd64
COMPILE_IMAGE?=pf9-compile
DOCKER?=docker
DOCKER_RUN_CMD?=${DOCKER} run
RM?=rm

ADDR_CONV_SRC_DIR=$(ROOT_DIR)/addr_conv
ADDR_CONV_BUILD_DIR=$(BUILD_DIR)/addr_conv
ADDR_CONV:=$(ADDR_CONV_BUILD_DIR)/addr_conv

.PHONY: addr-conv pf9kube-addr-conv clean
.DEFAULT_GOAL:= addr-conv

addr-conv:
	echo "make addr-conv"
	mkdir -p ${ADDR_CONV_BUILD_DIR}
	${DOCKER_RUN_CMD} --rm \
                        -e CGO_ENABLED=${CGO_ENABLED} \
                        -e GO111MODULE="on" \
                        -e GOOS=${GOOS} \
                        -e GOARCH=${GOARCH} \
                        -v ${ADDR_CONV_SRC_DIR}:/src:Z \
                        -v ${ADDR_CONV_BUILD_DIR}:/go/bin:Z \
                        ${COMPILE_IMAGE} \
                        sh -c 'cd /src && \
                        go mod download && \
                        go build -i -v -a -ldflags "-extldflags \"-static\"" \
                        -o /go/bin/addr_conv \
                        ./cmd/addr_conv';

pf9kube-addr-conv: addr-conv
	cp -a ${ADDR_CONV} ${AGENT_BUILD_DIR}

clean:
	${RM} -rf ./bin
