# IsoBox Quick Start Guide

Get started with IsoBox in 5 minutes.

## What You'll Learn

- How to initialize an IsoBox environment
- How to enter and use the isolated environment
- How to install packages
- How to manage multiple environments

## Prerequisites

- Linux or macOS system
- Go 1.20+ (for building from source)
- Optional: BusyBox for full POSIX compliance

## Installation

### Build from Source

```bash
git clone https://github.com/javanhut/isobox
cd isobox
go build -o isobox
sudo mv isobox /usr/local/bin/
```

### Verify Installation

```bash
isobox
```

You should see the usage information.

### Install BusyBox (Recommended)

For full POSIX compliance, install BusyBox:

**Ubuntu/Debian:**
```bash
sudo apt-get install busybox-static
```

**Alpine:**
```bash
apk add busybox-static
```

**macOS:**
```bash
brew install busybox
```

**Arch:**
```bash
sudo pacman -S busybox
```

## Your First Environment

### 1. Create a Project Directory

```bash
mkdir ~/my-first-isobox
cd ~/my-first-isobox
```

### 2. Initialize IsoBox

```bash
isobox init
```

You'll see output like:

```
Initializing IsoBox environment in: /home/user/my-first-isobox
  Created: .isobox
  Created: .isobox/bin
  Created: .isobox/lib
  Created: .isobox/etc
  Created: .isobox/var/lib/ipkg
  Created: .isobox/var/log
  Created: .isobox/tmp

Setting up POSIX environment...
Using system busybox: /usr/bin/busybox
  Installed busybox and applets
  Created environment configuration

IsoBox environment created successfully!
Location: /home/user/my-first-isobox

To enter the environment, run:
  cd /home/user/my-first-isobox && isobox enter
```

### 3. Check What Was Created

```bash
ls -la
```

You'll see a `.isobox` directory:

```
drwxr-xr-x .isobox
```

### 4. Enter the Environment

```bash
isobox enter
```

Your prompt changes to indicate you're in IsoBox:

```
(isobox) user@hostname:~/my-first-isobox$
```

### 5. Try Some Commands

Inside the IsoBox environment:

```bash
ls -la
pwd
whoami
echo $ISOBOX_ROOT
which ls
```

### 6. Exit the Environment

```bash
exit
```

You're back to your normal shell.

## Installing Packages

### 1. Enter Your Environment

```bash
cd ~/my-first-isobox
isobox enter
```

### 2. Install a Package

From within the environment OR from outside:

```bash
isobox pkg install curl
```

Output:
```
Installing package: curl
Downloading from Alpine repository...
Successfully installed curl
```

### 3. Use the Installed Package

```bash
isobox exec curl --version
```

Or from within the environment:

```bash
isobox enter
curl --version
```

### 4. List Installed Packages

```bash
isobox pkg list
```

Output:
```
Installed packages:
  curl (latest) - Package curl from Alpine
```

### 5. Remove a Package

```bash
isobox pkg remove curl
```

## Managing Multiple Environments

Each directory can have its own IsoBox environment.

### Create Multiple Environments

```bash
mkdir ~/project-a
cd ~/project-a
isobox init
isobox pkg install git vim

mkdir ~/project-b
cd ~/project-b
isobox init
isobox pkg install curl wget
```

### Switch Between Environments

```bash
cd ~/project-a
isobox enter
(isobox) $ which vim
/home/user/project-a/.isobox/bin/vim
(isobox) $ exit

cd ~/project-b
isobox enter
(isobox) $ which curl
/home/user/project-b/.isobox/bin/curl
(isobox) $ exit
```

Each environment is completely independent.

## Environment Status

Check the status of your environment:

```bash
cd ~/my-first-isobox
isobox status
```

Output:
```
IsoBox Environment Status
=========================

Root Directory: /home/user/my-first-isobox
Created: 2024-01-01 12:00:00
Binary Path: /home/user/my-first-isobox/.isobox/bin
Available Commands: 150
Installed Packages: 2

To enter this environment:
  cd /home/user/my-first-isobox && isobox enter
```

## Executing Commands Without Entering

You can execute commands in the environment without entering a shell:

```bash
cd ~/my-first-isobox
isobox exec ls -la
isobox exec pwd
isobox exec curl https://example.com
```

## Common Workflows

### Development Workflow

```bash
mkdir ~/myapp
cd ~/myapp

isobox init

isobox pkg install git
isobox pkg install make
isobox pkg install gcc

isobox enter

git clone https://github.com/user/repo .
make build

exit
```

### Testing Different Versions

```bash
mkdir ~/test-env-1
cd ~/test-env-1
isobox init
isobox pkg install python3

mkdir ~/test-env-2
cd ~/test-env-2
isobox init
isobox pkg install python3
```

### Quick Script Environment

```bash
mkdir ~/scripts
cd ~/scripts
isobox init
isobox pkg install bash coreutils grep sed awk

cat > script.sh << 'EOF'
#!/bin/bash
echo "Running in IsoBox"
ls -la | grep "isobox"
EOF

chmod +x script.sh
isobox exec ./script.sh
```

## Directory Structure

After initialization, your directory contains:

```
my-first-isobox/
├── .isobox/
│   ├── bin/              # POSIX binaries (150+ commands)
│   │   ├── busybox
│   │   ├── sh
│   │   ├── bash
│   │   ├── ls
│   │   └── ...
│   ├── lib/              # Libraries
│   ├── etc/              # Configuration
│   │   ├── profile       # Shell environment setup
│   │   └── environment   # Environment variables
│   ├── var/
│   │   └── lib/ipkg/
│   │       └── installed.json  # Package database
│   └── config.json       # Environment metadata
└── [your files]
```

## Cleaning Up

### Remove a Single Environment

```bash
cd ~/my-first-isobox
isobox destroy
```

This removes the `.isobox` directory but keeps your files.

### Manual Cleanup

```bash
rm -rf ~/my-first-isobox/.isobox
```

## Troubleshooting

### "busybox not found"

If BusyBox isn't available, IsoBox falls back to creating symlinks to system binaries. Install BusyBox for best results:

```bash
sudo apt-get install busybox-static
```

### "environment not found"

Make sure you're in a directory with an initialized IsoBox:

```bash
ls -la | grep isobox
```

If `.isobox` doesn't exist, run:

```bash
isobox init
```

### Commands Not Found in Environment

Check if the command is in the environment:

```bash
isobox exec which <command>
```

If not found, it might need to be installed:

```bash
isobox pkg install <package>
```

### Package Download Fails

Check internet connectivity:

```bash
curl https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/
```

If offline, you can't download packages. Use symlinks to system binaries instead (done automatically).

## Next Steps

Now that you know the basics:

1. Read [ARCHITECTURE.md](ARCHITECTURE.md) to understand how IsoBox works
2. Read [PACKAGE_MANAGER.md](PACKAGE_MANAGER.md) to learn about IPKG
3. Create project-specific environments
4. Explore available packages: https://pkgs.alpinelinux.org/packages

## Tips

- IsoBox environments are directory-scoped - one per project
- You can have unlimited environments
- Environments are independent - changes in one don't affect others
- `.isobox` is in `.gitignore` by default (don't commit it)
- BusyBox provides 150+ Unix commands in a single binary
- Packages come from Alpine Linux repository (small, secure)

## Example Projects

### Web Development

```bash
mkdir ~/webapp
cd ~/webapp
isobox init
isobox pkg install curl wget
isobox enter
```

### System Administration

```bash
mkdir ~/sysadmin
cd ~/sysadmin
isobox init
isobox pkg install bash grep sed awk
isobox enter
```

### Build Environment

```bash
mkdir ~/build
cd ~/build
isobox init
isobox pkg install gcc make
isobox enter
```

Happy IsoBoxing!
