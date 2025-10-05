.PHONY: build, install, uninstall, clean, check

check:
	go test

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o isobox


install:
	@echo "Installing ISOBOX"
	CGO_ENABLED=0 go build -ldflags="-s -w" -o isobox
	sudo mv isobox /usr/local/bin/
	@echo "Installed isobox to /usr/local/bin/"


uninstall:
	sudo rm -f /usr/local/bin/isobox
	@echo "Uninstalled isobox"


clean:
	rm -f isobox
