all:
	go install -v github.com/frankbraun/mole/...

.PHONY: test
test:
	go get github.com/frankbraun/gocheck
	gocheck -g -c
