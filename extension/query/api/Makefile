.PHONY: genall
genall:
	cp -rf opentelemetry-proto/opentelemetry ./
	hack/genproto.sh
	hack/genswagger.sh
	hack/gen-grpc-ts.sh
	rm -rf ./opentelemetry

.PHONY: clean-proto
clean-proto:
	hack/clean.sh proto

.PHONY: clean-swagger
clean-swagger:
	hack/clean.sh swagger

.PHONY: clean-sdk
clean-sdk:
	hack/clean.sh sdk

.PHONY: clean-all
clean-all:
	hack/clean.sh all

.PHONY: genproto
genproto:
	hack/genproto.sh

.PHONY: genswagger
genswagger:
	hack/genswagger.sh

.PHONY: gen-grpc-ts
gen-grpc-ts:
	hack/gen-grpc-ts.sh

.PHONY: push-grpc-ts
push-grpc-ts:
	hack/push-grpc-ts.sh

.PHONY: verify-swagger
verify-swagger:
	hack/verify-swagger.sh

.PHONY: verify-proto
verify-proto:
	hack/verify-proto.sh

.PHONY: verify-grpc-ts
verify-grpc-ts:
	hack/verify-grpc-ts.sh

.PHONY: verify
verify:
	hack/verify-all.sh

.PHONY: install_grpc_dep
install_grpc_dep:
	go env -w GOPROXY=https://goproxy.cn,direct && \
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.10.2 &&\
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.10.2 &&\
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0 &&\
	go install github.com/mwitkow/go-proto-validators/protoc-gen-govalidators@v0.3.2 &&\
	go install github.com/rakyll/statik@v0.1.7 &&\
	go install github.com/golang/protobuf/protoc-gen-go@v1.5.2 &&\
	go install github.com/grpc-ecosystem/protoc-gen-grpc-gateway-ts@v1.1.2 &&\
	go install github.com/ckaznocha/protoc-gen-lint@v0.2.4

