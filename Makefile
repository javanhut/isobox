.PHONY: build, install, uninstall, clean, check

check:
	go test

build:
	go build -o isobox


install:
	@echo "Installing ISOBOX"
	go build -o isobox
	sudo mkdir -p /usr/local/share/isobox/scripts
	sudo cp scripts/isobox-internal.sh /usr/local/share/isobox/scripts/
	sudo chmod +x /usr/local/share/isobox/scripts/isobox-internal.sh
	sudo mv isobox /usr/local/bin/
	@echo "Installed isobox to /usr/local/bin/"
	@echo "Installed scripts to /usr/local/share/isobox/scripts/"


uninstall:
	sudo rm -f /usr/local/bin/isobox
	sudo rm -rf /usr/local/share/isobox
	@echo "Uninstalled isobox"


clean:
	rm -f isobox
