.PHONY: ttr

ttr:
	goimports -w .
	go build
