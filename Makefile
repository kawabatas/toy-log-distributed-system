GO?=go

.PHONY: test
## test: runs go test
test:
	${GO} test -race ./...

.PHONY: gencert
## gencert: generates cert files
gencert:
	cfssl gencert \
		-initca testdata/ca-csr.json | cfssljson -bare ca
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=testdata/ca-config.json \
		-profile=server \
		testdata/server-csr.json | cfssljson -bare server
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=testdata/ca-config.json \
		-profile=client \
		-cn="root" \
		testdata/client-csr.json | cfssljson -bare root-client
	mv *.pem *.csr testdata/cert

.PHONY: help
## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'
