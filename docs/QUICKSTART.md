# IsoBox Quick Start Guide

Get started with IsoBox in 5 minutes.

## What You'll Learn

- How to initialize an IsoBox environment
- How to enter and use the isolated environment
- How to install packages
- How to manage multiple environments

## Prerequisites

- Linux system (chroot is Linux-specific)
- Go 1.20+ (for building from source)
- sudo access (required for chroot)
- BusyBox (recommended, will be installed if not present)

## Installation

### Build from Source

```bash
git clone https://github.com/javanhut/isobox
cd isobox
make install
```

This installs:
- Binary to `/usr/local/bin/isobox`
- Scripts to `/usr/local/share/isobox/scripts/`

### Verify Installation

```bash
isobox
```

You should see the usage information.

### Install BusyBox (Recommended)

For best results, install BusyBox on your host system:

**Ubuntu/Debian:**
```bash
sudo apt-get install busybox-static
```

**Arch:**
```bash
sudo pacman -S busybox
```

**Alpine:**
```bash
apk add busybox-static
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

By default, IsoBox uses `bash` as the shell. You can specify a different shell using the `--shell` flag:

```bash
isobox init --shell zsh    # Use zsh as default shell
isobox init --shell bash   # Use bash as default shell (default)
isobox init --shell sh     # Use sh as default shell
```

Available shells: `bash` (default), `zsh`, `sh`

You'll see output like:

```
Initializing IsoBox environment in: .
Creating isolated Linux filesystem...
  Created: .isobox/bin
  Created: .isobox/lib
  Created: .isobox/etc
  Created: .isobox/var/lib/ipkg

Setting up POSIX binaries...
Found BusyBox at: /usr/bin/busybox
  Added SSL certificates
  Added Alpine wget
  Added Alpine ca-certificates
  Added musl libc (for Alpine packages)
  Added Alpine base dependencies
  Installed BusyBox and 150+ Unix utilities

IsoBox environment created successfully!
Location: /home/user/my-first-isobox

To enter the environment, run:
  cd . && isobox enter
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

You'll see a welcome banner:

```
=========================================
   ISOBOX Isolated Environment
=========================================
You are in a completely isolated Linux
environment. You CANNOT access the host
system from here.

Package Manager:
  isobox install <package>
  isobox remove <package>
  isobox list

Type 'exit' to leave this environment
=========================================

(isobox) root@hostname:/$
```

### 5. Try Some Commands

Inside the IsoBox environment:

```bash
ls -la
pwd
whoami
cat /etc/os-release
which ls
```

### 6. Exit the Environment

```bash
exit
```

You're back to your normal shell.

## Installing Packages

All package management happens **inside** the isolated environment.

### 1. Enter Your Environment

```bash
cd ~/my-first-isobox
isobox enter
```

### 2. Install a Package

From within the environment:

```bash
(isobox) # isobox install git
```

Output:
```
Installing git (host: Arch Linux)...
Downloading from Alpine Linux repository...
Fetching package list...
Downloading git-2.43.7-r0.apk...
Extracting package...
Installing Alpine library dependencies...
  Installing pcre2...
Successfully installed git
```

### 3. Use the Installed Package

```bash
(isobox) # git --version
git version 2.43.7
```

### 4. List Installed Packages

```bash
(isobox) # isobox list
```

Output:
```
Installed packages:
  git (latest) - installed 2025-10-03T14:00:00Z
```

### 5. Remove a Package

```bash
(isobox) # isobox remove git
```

## Managing Multiple Environments

Each directory can have its own IsoBox environment.

### Create Multiple Environments

```bash
mkdir ~/project-a
cd ~/project-a
isobox init
isobox enter
(isobox) # isobox install git
(isobox) # isobox install vim
(isobox) # exit

mkdir ~/project-b
cd ~/project-b
isobox init
isobox enter
(isobox) # isobox install curl
(isobox) # isobox install wget
(isobox) # exit
```

### Switch Between Environments

```bash
cd ~/project-a
isobox enter
(isobox) # which git
/usr/bin/git
(isobox) # exit

cd ~/project-b
isobox enter
(isobox) # which git
(command not found)
(isobox) # which curl
/usr/bin/curl
(isobox) # exit
```

Each environment is completely independent.

## Environment Status

Check the status of your environment from the host:

```bash
cd ~/my-first-isobox
isobox status
```

Output:
```
ISOBOX Environment Status
=========================

Project Root: /home/user/my-first-isobox
Isolated Root: /home/user/my-first-isobox/.isobox
Created: 2025-10-03 14:00:00
Available Commands: 150
Shared Libraries: 45
Installed Packages: 2

This is a COMPLETELY ISOLATED environment.
The host system is NOT accessible from within.

To enter:
  cd /home/user/my-first-isobox && isobox enter
```

## Executing Commands Without Entering

You can execute commands in the environment without entering a shell:

```bash
cd ~/my-first-isobox
isobox exec ls -la /
isobox exec pwd
isobox exec cat /etc/os-release
```

## Common Workflows

### Development Workflow

```bash
mkdir ~/myapp
cd ~/myapp

isobox init
isobox enter

(isobox) # isobox install git
(isobox) # isobox install make
(isobox) # git clone https://github.com/user/repo .
(isobox) # make build

(isobox) # exit
```

### Testing Environment

```bash
mkdir ~/test-env
cd ~/test-env

isobox init
isobox enter

(isobox) # isobox install python3
(isobox) # python3 --version

(isobox) # exit
```

### Quick Script Environment

```bash
mkdir ~/scripts
cd ~/scripts

isobox init
isobox enter

(isobox) # cat > script.sh << 'EOF'
#!/bin/sh
echo "Running in IsoBox"
ls -la | grep "isobox"
EOF

(isobox) # chmod +x script.sh
(isobox) # ./script.sh

(isobox) # exit
```

## Directory Structure

After initialization, your directory contains:

```
my-first-isobox/
├── .isobox/
│   ├── bin/              # POSIX binaries (BusyBox + symlinks)
│   │   ├── busybox
│   │   ├── sh -> busybox
│   │   ├── ls -> busybox
│   │   ├── cat -> busybox
│   │   └── ...
│   ├── usr/
│   │   └── bin/
│   │       ├── wget      # Alpine wget (SSL capable)
│   │       └── git       # Installed packages
│   ├── lib/              # musl libc and libraries
│   │   ├── ld-musl-x86_64.so.1
│   │   └── libc.musl-x86_64.so.1
│   ├── usr/lib/          # Additional libraries
│   ├── etc/              # Configuration
│   │   ├── passwd
│   │   ├── group
│   │   ├── hosts
│   │   ├── resolv.conf
│   │   ├── bash.bashrc
│   │   ├── profile
│   │   └── os-release
│   ├── var/
│   │   └── lib/ipkg/
│   │       └── installed.json  # Package database
│   └── config.json       # Environment metadata
└── [your files]
```

## Advanced Features

### Choosing Your Shell

IsoBox supports multiple shells that are cached during initial installation for fast access:

```bash
# Initialize with bash (default)
isobox init --shell bash

# Initialize with zsh
isobox init --shell zsh

# Initialize with sh (BusyBox)
isobox init --shell sh
```

The selected shell becomes the default login shell when you enter the environment. All shells (bash, zsh, sh) are available in the environment regardless of which one you choose as default.

To use a different shell after initialization, you can manually run it inside the environment:

```bash
isobox enter
(isobox) # zsh        # Switch to zsh
(isobox) # bash       # Switch to bash
(isobox) # sh         # Switch to sh
```

### Migrating Directories Into IsoBox

You can copy directories from your host system into the isolated environment:

```bash
cd ~/my-first-isobox

# Copy a project directory into the isobox
isobox migrate ./mycode /home/username/mycode

# Enter and access the migrated files
isobox enter
(isobox) # cd /home/username/mycode
(isobox) # ls -la
(isobox) # exit
```

The `migrate` command:
- Copies files from host to the isolated filesystem
- Sets proper ownership (UID 1000)
- Makes files accessible inside the environment
- Does NOT affect the host files

### Rebuilding the Base System Cache

IsoBox caches the base system at `~/.cache/isobox/base-system.tar.gz` for faster initialization. If the cache becomes corrupted or you need to rebuild it:

```bash
# Rebuild the cache
isobox recache
```

This will:
- Delete the old cache
- Build a new base system from scratch
- Include the latest internal package manager script
- Cache it for future `isobox init` calls

After rebuilding the cache, you may want to destroy and reinitialize existing environments:

```bash
cd ~/my-first-isobox
isobox destroy
isobox init
```

## Cleaning Up

### Remove a Single Environment

```bash
cd ~/my-first-isobox
isobox destroy
```

This removes the `.isobox` directory but keeps your files. Uses `sudo` to handle root-owned files created during chroot operations.

### Manual Cleanup (if needed)

```bash
sudo rm -rf ~/my-first-isobox/.isobox
```

## Troubleshooting

### "BusyBox not found"

Install BusyBox on your host system:

```bash
# Arch
sudo pacman -S busybox

# Ubuntu/Debian
sudo apt-get install busybox-static
```

Then reinitialize:

```bash
isobox destroy
isobox init
```

### "Environment not found"

Make sure you're in a directory with an initialized IsoBox:

```bash
ls -la | grep isobox
```

If `.isobox` doesn't exist, run:

```bash
isobox init
```

### "Command not found" inside environment

Enter the environment and install the package:

```bash
isobox enter
(isobox) # isobox install <package>
```

### Library errors

If you see "symbol not found" or library errors:

```bash
isobox destroy
isobox init
```

This rebuilds the environment with the latest Alpine dependencies.

### Package download fails

Check internet connectivity and that you can reach Alpine repos:

```bash
ping dl-cdn.alpinelinux.org
```

Make sure port 443 (HTTPS) is not blocked.

### "isobox: not found" inside environment

If you get `/bin/sh: isobox: not found` when trying to install packages inside the environment:

```bash
# Exit the environment first
exit

# Rebuild the base system cache
isobox recache

# Destroy and reinitialize your environment
cd ~/my-first-isobox
isobox destroy
isobox init

# Enter and try again
isobox enter
(isobox) # isobox install git
```

This ensures the internal package manager script is properly installed.

## Next Steps

Now that you know the basics:

1. Read [ARCHITECTURE.md](ARCHITECTURE.md) to understand how IsoBox works
2. Read [PACKAGE_MANAGER.md](PACKAGE_MANAGER.md) to learn about the internal package manager
3. Create project-specific environments
4. Explore available packages: https://pkgs.alpinelinux.org/packages

## Tips

- IsoBox environments are directory-scoped - one per project
- You can have unlimited environments
- Environments are independent - changes in one don't affect others
- Don't commit `.isobox` to version control
- BusyBox provides 150+ Unix commands in a single binary
- Packages come from Alpine Linux repository (small, secure, musl-based)
- All package management happens inside the environment
- Libraries are automatically resolved from Alpine repos

## Example Projects

### Web Development

```bash
mkdir ~/webapp
cd ~/webapp
isobox init
isobox enter
(isobox) # isobox install curl
(isobox) # isobox install wget
(isobox) # isobox install nodejs
```

### System Administration

```bash
mkdir ~/sysadmin
cd ~/sysadmin
isobox init
isobox enter
(isobox) # isobox install bash
(isobox) # isobox install grep
(isobox) # isobox install sed
(isobox) # isobox install awk
```

### Python Development

```bash
mkdir ~/pyproject
cd ~/pyproject
isobox init
isobox enter
(isobox) # isobox install python3
(isobox) # isobox install py3-pip
(isobox) # python3 --version
```

Happy IsoBoxing!
