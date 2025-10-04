# IsoBox Internal Package Manager

The IsoBox package manager runs **inside** the isolated environment and installs packages from Alpine Linux repositories. All package management happens from within the chroot environment, not from the host system.

## Overview

The package manager is a shell script (`/bin/isobox`) that runs inside each isolated environment. It downloads Alpine Linux packages and extracts them directly into the isolated filesystem.

**Key Features:**
- Runs inside the isolated environment
- Installs packages from Alpine Linux v3.19 repositories
- Automatic dependency installation for common libraries
- Per-environment package isolation
- Simple JSON-based package tracking
- Supports both main and community repositories

## Basic Usage

All package commands run **inside** the isolated environment.

### Enter the Environment

```bash
cd ~/myproject
isobox enter
```

You're now inside the isolated environment with the `(isobox)` prompt.

### Install a Package

```bash
(isobox) # isobox install <package-name>
```

Examples:
```bash
(isobox) # isobox install git
(isobox) # isobox install curl
(isobox) # isobox install vim
(isobox) # isobox install python3
```

### Remove a Package

```bash
(isobox) # isobox remove <package-name>
```

Example:
```bash
(isobox) # isobox remove git
```

Note: This removes the package from the database but does not delete files.

### List Installed Packages

```bash
(isobox) # isobox list
```

Output:
```
Installed packages:
  git (latest) - installed 2025-10-03T14:00:00Z
  curl (latest) - installed 2025-10-03T14:15:00Z
```

### Update Package Index

```bash
(isobox) # isobox update
```

This displays information about updating the package index.

### Show Help

```bash
(isobox) # isobox help
```

## How It Works

### Package Sources

The package manager uses Alpine Linux v3.19 repositories:

**Main Repository:** https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/
**Community Repository:** https://dl-cdn.alpinelinux.org/alpine/v3.19/community/x86_64/

Alpine packages are used because:
- **Small size**: Built with musl libc (smaller than glibc)
- **Security**: Security-focused distribution
- **Compatibility**: Works in isolated environments with musl
- **Complete**: Large package selection across main and community repos
- **Actively maintained**: Regular updates and security patches

### Installation Process

1. **Check if installed**: Query the package database
2. **Search repositories**: Look in main repo, then community repo if not found
3. **Download package**: Fetch .apk file using wget or curl
4. **Extract package**: Unpack directly to `/` (which is the isolated root)
5. **Install dependencies**: Auto-install pcre2 and zlib on first package install
6. **Update database**: Add package entry to `/var/lib/ipkg/installed.json`
7. **Cleanup**: Remove downloaded .apk file

### Automatic Dependency Installation

During initialization (`isobox init`), the following are automatically installed:

**Core Libraries:**
- musl libc (required for all Alpine packages)
- libssl3 and libcrypto3 (SSL/TLS support)
- ca-certificates-bundle (SSL certificates)
- zlib (compression library)
- pcre2 (regex library)
- libidn2 and libunistring (internationalization)
- ncurses libraries (libncursesw, libformw, libmenuw, libpanelw)
- readline (command-line editing)
- libacl and libattr (ACL support)
- oniguruma (regex engine)
- libcap (capability support)
- s6 and skalibs (process supervision)
- utmps-libs (utmp/wtmp handling)

**Tools:**
- BusyBox (150+ Unix commands)
- Alpine wget (SSL-capable for package downloads)
- bash and zsh shells
- python3 (Python 3 interpreter)
- gcc (C compiler)
- go (Go programming language)
- vim (text editor)

When you install your first package, the package manager also installs common dependencies (pcre2, zlib) if not already marked as installed.

### Package Database

**Location:** `/var/lib/ipkg/installed.json`

This is inside the isolated environment, so from the host it's at `.isobox/var/lib/ipkg/installed.json`.

**Format:**
```json
[
  {
    "name": "git",
    "version": "latest",
    "installed": "2025-10-03T14:00:00Z"
  },
  {
    "name": "curl",
    "version": "latest",
    "installed": "2025-10-03T14:15:00Z"
  }
]
```

## Environment-Specific Packages

Each IsoBox environment has its own package manager and database:

```
project-a/
└── .isobox/
    ├── bin/isobox              # Package manager script
    └── var/lib/ipkg/
        └── installed.json      # Contains: git, vim

project-b/
└── .isobox/
    ├── bin/isobox              # Separate package manager
    └── var/lib/ipkg/
        └── installed.json      # Contains: python3, curl
```

Installing packages in `project-a` doesn't affect `project-b`.

## Common Packages

### Development Tools

```bash
(isobox) # isobox install git
(isobox) # isobox install make
(isobox) # isobox install gcc
(isobox) # isobox install cmake
```

### Networking Tools

```bash
(isobox) # isobox install curl
(isobox) # isobox install wget
(isobox) # isobox install openssh-client
(isobox) # isobox install netcat-openbsd
```

### Text Editors

Note: vim is pre-installed in all environments.

```bash
(isobox) # vim --version        # Already available
(isobox) # isobox install nano
(isobox) # isobox install emacs
```

### Programming Languages

Note: python3 and go are pre-installed in all environments.

```bash
(isobox) # python3 --version   # Already available
(isobox) # go version           # Already available
(isobox) # isobox install nodejs
(isobox) # isobox install ruby
```

### Text Processing

```bash
(isobox) # isobox install grep
(isobox) # isobox install sed
(isobox) # isobox install gawk
(isobox) # isobox install jq
```

### Build Tools

Note: gcc is pre-installed in all environments.

```bash
(isobox) # gcc --version        # Already available
(isobox) # isobox install g++
(isobox) # isobox install make
(isobox) # isobox install cmake
(isobox) # isobox install autoconf
```

## Complete Workflows

### Development Environment

```bash
# From host
mkdir ~/myproject
cd ~/myproject
isobox init
isobox enter

# Inside environment - vim, gcc, python3, go are pre-installed
(isobox) # vim --version
(isobox) # gcc --version
(isobox) # python3 --version
(isobox) # go version

# Install additional tools
(isobox) # isobox install git
(isobox) # isobox install make

# Use the tools
(isobox) # git --version
(isobox) # exit
```

### Build Environment

```bash
# From host
mkdir ~/build-env
cd ~/build-env
isobox init
isobox enter

# Inside environment - gcc is pre-installed
(isobox) # gcc --version
(isobox) # isobox install make
(isobox) # isobox install cmake
(isobox) # isobox install git
(isobox) # exit
```

### Python Development

```bash
# From host
mkdir ~/pyproject
cd ~/pyproject
isobox init
isobox enter

# Inside environment - python3 and gcc are pre-installed
(isobox) # python3 --version
(isobox) # isobox install py3-pip
(isobox) # isobox install python3-dev
(isobox) # pip --version
(isobox) # exit
```

### Web Development

```bash
# From host
mkdir ~/webapp
cd ~/webapp
isobox init
isobox enter

# Inside environment
(isobox) # isobox install nodejs
(isobox) # isobox install npm
(isobox) # isobox install git
(isobox) # isobox install curl
(isobox) # node --version
(isobox) # npm --version
(isobox) # exit
```

## Finding Packages

Search Alpine packages online:
https://pkgs.alpinelinux.org/packages

Search for a specific package:
```
https://pkgs.alpinelinux.org/packages?name=<package>&branch=v3.19
```

Examples:
- https://pkgs.alpinelinux.org/packages?name=git&branch=v3.19
- https://pkgs.alpinelinux.org/packages?name=python3&branch=v3.19
- https://pkgs.alpinelinux.org/packages?name=nodejs&branch=v3.19

## Repository Search Order

When you install a package, the package manager:

1. Searches the **main** repository first
2. If not found, searches the **community** repository
3. If not found in either, reports an error

Example:
```bash
(isobox) # isobox install git
Installing git (host: Arch Linux)...
Downloading from Alpine Linux repository...
Fetching package list...
Downloading git-2.43.7-r0.apk...
Extracting package...
Successfully installed git
```

If package is in community repo:
```bash
(isobox) # isobox install nodejs
Installing nodejs (host: Arch Linux)...
Downloading from Alpine Linux repository...
Fetching package list...
Warning: Package not found in main repo, trying community repo...
Downloading nodejs-20.10.0-r1.apk...
Extracting package...
Successfully installed nodejs
```

## Dependency Management

### Automatic Dependencies

The package manager automatically handles:

**During `isobox init`:**
- musl libc (required for all Alpine packages)
- SSL libraries (libssl3, libcrypto3)
- CA certificates (ca-certificates-bundle)
- Base libraries (zlib, pcre2, libidn2, libunistring)

**During first package install:**
- pcre2 (regex support)
- zlib (compression support)

These are installed once and marked with a flag file at `/var/lib/ipkg/.alpine_deps_installed`.

### Package-Specific Dependencies

Some packages have specific dependencies you need to install manually:

**Python development:**
```bash
(isobox) # isobox install python3
(isobox) # isobox install py3-pip
(isobox) # isobox install python3-dev
(isobox) # isobox install gcc
(isobox) # isobox install musl-dev
```

**Node.js development:**
```bash
(isobox) # isobox install nodejs
(isobox) # isobox install npm
```

**Git with SSH:**
```bash
(isobox) # isobox install git
(isobox) # isobox install openssh-client
```

## Package Database Operations

### View Database from Host

```bash
cat .isobox/var/lib/ipkg/installed.json
```

### View Database from Inside Environment

```bash
(isobox) # cat /var/lib/ipkg/installed.json
```

### Format with jq

If jq is installed:
```bash
(isobox) # isobox install jq
(isobox) # cat /var/lib/ipkg/installed.json | jq
```

Output:
```json
[
  {
    "name": "git",
    "version": "latest",
    "installed": "2025-10-03T14:00:00Z"
  }
]
```

### Manual Database Editing

Not recommended, but possible:
```bash
(isobox) # vi /var/lib/ipkg/installed.json
```

The database is plain JSON, so you can manually add or remove entries if needed.

## Troubleshooting

### Package Not Found

**Error:** `Error: Package <name> not found in Alpine repositories`

**Causes:**
- Package name is misspelled
- Package doesn't exist in Alpine v3.19
- Package is in a different repository (testing, edge)

**Solutions:**
1. Check spelling of package name
2. Search Alpine package database: https://pkgs.alpinelinux.org/packages
3. Try alternative names (e.g., `python3` not `python`)
4. Check if package exists in v3.19 specifically

### Download Failed

**Error:** `Error: Failed to download package`

**Causes:**
- No internet connection
- Repository is down
- DNS not working inside environment

**Solutions:**
1. Check internet from host: `ping dl-cdn.alpinelinux.org`
2. Check DNS inside environment: `(isobox) # cat /etc/resolv.conf`
3. Verify wget/curl is working: `(isobox) # wget --version`
4. Try again later if repository is temporarily down

### Extract Failed

**Error:** `Error: Failed to extract package`

**Causes:**
- Corrupted download
- Insufficient permissions
- tar not available

**Solutions:**
1. Try reinstalling: `(isobox) # isobox remove <pkg> && isobox install <pkg>`
2. Check available space: `(isobox) # df -h`
3. Verify tar is available: `(isobox) # which tar`

### wget or curl Not Found

**Error:** `Error: wget or curl required for package installation`

**Cause:** Both wget and curl are missing from the environment

**Solution:**
This should not happen after `isobox init` because Alpine wget is automatically installed. If it does:

```bash
# From host
cd ~/myproject
isobox destroy
isobox init
```

This rebuilds the environment with all required tools.

### Package Already Installed

**Message:** `Package <name> is already installed`

**Cause:** Package exists in database

**Solution:**
If you want to reinstall:
```bash
(isobox) # isobox remove <package>
(isobox) # isobox install <package>
```

### Library Errors After Install

**Error:** `error while loading shared libraries: libfoo.so.1`

**Cause:** Package depends on libraries not in the base dependencies

**Solution:**
1. Identify missing library: `(isobox) # ldd /usr/bin/<binary>`
2. Search for package providing library: https://pkgs.alpinelinux.org/contents
3. Install the library package: `(isobox) # isobox install <library-package>`

Common library packages:
- `libssl3` - SSL/TLS
- `libcrypto3` - Cryptography
- `pcre2` - Regular expressions
- `zlib` - Compression
- `ncurses-libs` - Terminal UI
- `readline` - Command-line editing

### Package Manager Not Found

**Error:** `/bin/sh: isobox: not found` when running package commands inside the environment

**Cause:** The package manager script (`/bin/isobox`) is missing from the isolated environment. This can happen if:
- The cache was created before the script was properly included
- The environment was created with an older version of IsoBox
- The base system cache is corrupted

**Solution:**

```bash
# Exit the environment if you're inside it
exit

# Rebuild the base system cache from the host
isobox recache

# Destroy and reinitialize your environment
cd ~/myproject
isobox destroy
isobox init

# Enter and verify the package manager works
isobox enter
(isobox) # isobox help
(isobox) # isobox install git
```

The `recache` command:
- Deletes the old base system cache at `~/.cache/isobox/base-system.tar.gz`
- Rebuilds it from scratch with the latest package manager script
- Ensures all future `isobox init` commands use the updated cache

## Technical Details

### Package Manager Script Location

Inside environment: `/bin/isobox`
From host: `.isobox/bin/isobox`

This is a POSIX shell script that runs inside the chroot environment.

### APK File Format

Alpine packages (.apk) are gzip-compressed tar archives containing:

```
package-name-version.apk
├── .PKGINFO          # Package metadata
├── .trigger          # Installation triggers (optional)
├── usr/
│   ├── bin/          # Executables
│   ├── lib/          # Libraries
│   └── share/        # Data files
└── etc/              # Configuration files
```

The package manager extracts the entire archive to `/` (the isolated root).

### Download URL Structure

```
https://dl-cdn.alpinelinux.org/alpine/v3.19/{repo}/x86_64/{package}.apk
```

Components:
- `v3.19` - Alpine version
- `{repo}` - Repository: `main` or `community`
- `x86_64` - Architecture
- `{package}` - Package filename with version

Example:
```
https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/git-2.43.7-r0.apk
```

### Package Discovery

The script scrapes the repository index HTML:
1. Fetch repository index page with wget/curl
2. Parse HTML for package links matching pattern: `href="<package>-[0-9]`
3. Extract the first matching package filename
4. Construct full download URL
5. Download and extract

### Database Operations

All database operations use either jq (if available) or sed:

**Add package (with jq):**
```bash
jq ". += [{\"name\": \"$pkg\", \"version\": \"latest\", \"installed\": \"$date\"}]" installed.json
```

**Add package (with sed):**
```bash
sed -i 's/\]$/,{"name":"'$pkg'","version":"latest","installed":"'$date'"}]/' installed.json
```

**Remove package (with jq):**
```bash
jq "map(select(.name != \"$pkg\"))" installed.json
```

### Cache Location

Downloaded packages are temporarily stored at:
```
/var/cache/isobox/<package>.apk
```

This cache is inside the isolated environment, so from the host:
```
.isobox/var/cache/isobox/<package>.apk
```

Packages are deleted immediately after extraction to save space.

## Limitations

### 1. No Full Dependency Resolution

Only base libraries (pcre2, zlib) are auto-installed. Other dependencies must be installed manually.

Example:
```bash
# Wrong - may fail with missing dependencies
(isobox) # isobox install gcc

# Right - install dependencies first
(isobox) # isobox install musl-dev
(isobox) # isobox install binutils
(isobox) # isobox install gcc
```

### 2. No Version Pinning

Always installs the latest version from Alpine v3.19 repositories. You cannot specify a version.

### 3. No Package Verification

No GPG signature checking. Packages are downloaded over HTTPS but signatures are not verified.

### 4. Basic Removal

`isobox remove <package>` only removes the database entry. Files remain on disk.

To fully remove:
```bash
# From host
sudo rm -rf .isobox/usr/bin/<binary>
sudo rm -rf .isobox/usr/lib/<libraries>
```

### 5. No Search Command

No built-in package search. Use the Alpine package website:
https://pkgs.alpinelinux.org/packages

### 6. No Upgrade Command

No way to upgrade packages. To update a package:
```bash
(isobox) # isobox remove <package>
(isobox) # isobox install <package>
```

### 7. No Conflict Detection

If two packages provide the same file, the last one installed wins (files are overwritten).

### 8. Architecture Locked to x86_64

Only x86_64 packages are supported. No ARM, ARM64, or other architectures.

## Best Practices

### 1. Initialize Before Installing

Always run `isobox init` before entering the environment:
```bash
mkdir ~/myproject
cd ~/myproject
isobox init
isobox enter
```

### 2. Install Packages Inside Environment

Package commands only work inside the environment:
```bash
# Wrong - from host
isobox install git

# Right - from inside
isobox enter
(isobox) # isobox install git
```

### 3. Document Package Dependencies

Keep a list of packages in a script or README:
```bash
#!/bin/sh
# setup-env.sh
isobox install git
isobox install vim
isobox install make
isobox install gcc
```

### 4. One Environment Per Project

Don't share environments across projects:
```
~/project-a/.isobox    # Separate environment
~/project-b/.isobox    # Separate environment
```

### 5. Don't Commit .isobox/

Add to `.gitignore`:
```
.isobox/
```

Environments are easily recreated with `isobox init`.

### 6. Install Common Dependencies First

For development, install base tools first:
```bash
(isobox) # isobox install git
(isobox) # isobox install vim
(isobox) # isobox install make
```

Then install project-specific packages.

### 7. Check Package Availability

Before installing, verify the package exists:
```
https://pkgs.alpinelinux.org/packages?name=<package>&branch=v3.19
```

This prevents failed installations.

## Examples

### Full Python Environment

```bash
# From host
mkdir ~/pyproject
cd ~/pyproject
isobox init
isobox enter

# Inside environment - python3 and gcc are pre-installed
(isobox) # python3 --version
(isobox) # gcc --version
(isobox) # isobox install py3-pip
(isobox) # isobox install python3-dev
(isobox) # isobox install musl-dev

# Use Python
(isobox) # pip install requests
(isobox) # python3 -c "import requests; print(requests.__version__)"
(isobox) # exit
```

### Full Node.js Environment

```bash
# From host
mkdir ~/nodeapp
cd ~/nodeapp
isobox init
isobox enter

# Inside environment
(isobox) # isobox install nodejs
(isobox) # isobox install npm
(isobox) # isobox install git

# Use Node
(isobox) # node --version
(isobox) # npm --version
(isobox) # npm init -y
(isobox) # npm install express
(isobox) # exit
```

### System Administration Tools

```bash
# From host
mkdir ~/sysadmin
cd ~/sysadmin
isobox init
isobox enter

# Inside environment
(isobox) # isobox install bash
(isobox) # isobox install coreutils
(isobox) # isobox install findutils
(isobox) # isobox install grep
(isobox) # isobox install sed
(isobox) # isobox install gawk
(isobox) # isobox install curl
(isobox) # isobox install jq

# Use tools
(isobox) # bash --version
(isobox) # ls --version
(isobox) # grep --version
(isobox) # exit
```

### Build Environment for C/C++

```bash
# From host
mkdir ~/build
cd ~/build
isobox init
isobox enter

# Inside environment - gcc is pre-installed
(isobox) # gcc --version
(isobox) # isobox install g++
(isobox) # isobox install make
(isobox) # isobox install cmake
(isobox) # isobox install git
(isobox) # isobox install musl-dev

# Build a project
(isobox) # git clone https://github.com/user/project.git
(isobox) # cd project
(isobox) # make
(isobox) # exit
```

## Comparison with Other Package Managers

| Feature | IsoBox | apk | apt | npm | pip |
|---------|--------|-----|-----|-----|-----|
| Scope | Per-directory | System | System | Project | Project |
| Isolation | Complete | None | None | Partial | Partial |
| Dependencies | Semi-auto | Auto | Auto | Auto | Auto |
| Database | JSON | SQLite | dpkg | package.json | Metadata |
| Size | Tiny | Small | Large | Medium | Medium |
| Language | Shell | C | C | JavaScript | Python |
| Verification | None | GPG | GPG | SHA | SHA |

IsoBox package manager is designed for simplicity and environment isolation, not as a replacement for full-featured system package managers.

## Future Enhancements

Planned improvements:

1. **Full dependency resolution** - Automatically install all required packages
2. **GPG verification** - Verify package signatures
3. **Version pinning** - Install specific package versions
4. **Package search** - Built-in search: `isobox search <term>`
5. **Clean removal** - Delete files on `isobox remove`
6. **Upgrade command** - `isobox upgrade <package>`
7. **List available** - Show all available packages
8. **Package info** - Display package details
9. **Multiple architectures** - Support ARM, ARM64
10. **Custom repositories** - Add non-Alpine repositories

## Contributing

To improve the package manager:

1. Edit `scripts/isobox-internal.sh` (source script)
2. Test in a clean environment
3. Update this documentation
4. Submit a pull request

The script is installed to `.isobox/bin/isobox` during `isobox init`.
