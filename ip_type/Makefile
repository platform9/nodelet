
CGO_ENABLED?=0
GOOS?=linux
GOARCH?=amd64
COMPILE_IMAGE?=pf9-compile
DOCKER?=docker
DOCKER_RUN_CMD?=${DOCKER} run
RM?=rm

IP_TYPE_SRC_DIR=$(ROOT_DIR)/ip_type
IP_TYPE_BUILD_DIR=$(BUILD_DIR)/ip_type
IP_TYPE:=$(IP_TYPE_BUILD_DIR)/ip_type

.PHONY: ip_type pf9kube-ip_type clean
.DEFAULT_GOAL:= ip_type

ip_type:
	echo "make ip_type"
	mkdir -p ${IP_TYPE_BUILD_DIR}
	${DOCKER_RUN_CMD} --rm \
                        -e CGO_ENABLED=${CGO_ENABLED} \
                        -e GO111MODULE="on" \
                        -e GOOS=${GOOS} \
                        -e GOARCH=${GOARCH} \
                        -v ${IP_TYPE_SRC_DIR}:/src:Z \
                        -v ${IP_TYPE_BUILD_DIR}:/go/bin:Z \
                        ${COMPILE_IMAGE} \
                        sh -c 'cd /src && \
                        go mod download && \
                        go build -i -v -a -ldflags "-extldflags \"-static\"" \
                        -o /go/bin/ip_type \
                        ./cmd/main.go';

pf9kube-ip_type: ip_type
	cp -a ${IP_TYPE} ${AGENT_BUILD_DIR}

clean:
	${RM} -rf ./bin
