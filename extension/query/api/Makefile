.PHONY: genall
genall: install_tools
	buf generate

proto-lint:
	buf lint

.PHONY: install_tools
install_tools:
	# Substitute GOBIN for your bin directory
    # Leave unset to default to $GOPATH/bin
    GO111MODULE=on GOBIN=/usr/local/bin go install \
      github.com/bufbuild/buf/cmd/buf@v1.10.0

