.PHONY: ttr

ttr:
	goimports -w .
	go build
	sudo chown root:root ttr
	sudo chmod u+s ttr
