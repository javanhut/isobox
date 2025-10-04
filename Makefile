.PHONY: build, install, unistall, clean, check

check:
	go test

build:
	go build -o isobox


install:
	@echo "Installing ISOBOX"
	go build -o isobox && sudo mv isobox /usr/local/bin/


uninstall:
	sudo rm /usr/local/bin/isobox


clean:
	rm isobox
