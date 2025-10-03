# IPKG - IsoBox Package Manager

IPKG is the lightweight package manager for IsoBox environments, providing per-directory package installation from Alpine Linux repositories.

## Overview

IPKG manages packages within each IsoBox environment independently. Packages installed in one environment don't affect other environments or the host system.

**Key Features:**
- Install packages from Alpine Linux repository
- Per-environment package isolation
- Simple JSON-based tracking
- Minimal overhead

## Basic Usage

### Install a Package

```bash
isobox pkg install <package-name>
```

Examples:
```bash
isobox pkg install curl
isobox pkg install git
isobox pkg install vim
isobox pkg install python3
```

### Remove a Package

```bash
isobox pkg remove <package-name>
```

Example:
```bash
isobox pkg remove curl
```

### List Installed Packages

```bash
isobox pkg list
```

Output:
```
Installed packages:
  curl (8.5.0-r0) - Package curl from Alpine
  git (2.43.0-r0) - Package git from Alpine
  vim (9.0.2116-r0) - Package vim from Alpine
```

### Update Package Index

```bash
isobox pkg update
```

This updates the repository metadata (currently informational).

## How It Works

### Package Source

IPKG uses Alpine Linux packages as its source:

**Repository:** https://dl-cdn.alpinelinux.org/alpine/v3.19/main

Alpine packages are chosen because:
- **Small size**: Built with musl libc (smaller than glibc)
- **Security**: Security-focused distribution
- **Compatibility**: Works well in isolated environments
- **Active**: Well-maintained and updated
- **Complete**: Large package selection

### Installation Process

1. **Check**: Verify package isn't already installed
2. **Download**: Fetch APK file from Alpine repository
3. **Extract**: Unpack to `.isobox/` directory
4. **Register**: Add to package database
5. **Complete**: Binary available in environment

### Package Database

Located at: `.isobox/var/lib/ipkg/installed.json`

Format:
```json
[
  {
    "name": "curl",
    "version": "8.5.0-r0",
    "description": "Package curl from Alpine",
    "files": ["/usr/bin/curl"]
  }
]
```

## Environment-Specific Packages

Each IsoBox environment maintains its own package database:

```
project-a/
└── .isobox/
    └── var/lib/ipkg/installed.json    (curl, git)

project-b/
└── .isobox/
    └── var/lib/ipkg/installed.json    (vim, python3)
```

Installing a package in `project-a` doesn't affect `project-b`.

## Common Packages

### Development Tools

```bash
isobox pkg install git
isobox pkg install gcc
isobox pkg install make
isobox pkg install cmake
```

### Networking Tools

```bash
isobox pkg install curl
isobox pkg install wget
isobox pkg install openssh-client
isobox pkg install netcat-openbsd
```

### Text Editors

```bash
isobox pkg install vim
isobox pkg install nano
isobox pkg install emacs
```

### Programming Languages

```bash
isobox pkg install python3
isobox pkg install nodejs
isobox pkg install ruby
isobox pkg install go
```

### Text Processing

```bash
isobox pkg install grep
isobox pkg install sed
isobox pkg install awk
isobox pkg install jq
```

### Build Tools

```bash
isobox pkg install gcc
isobox pkg install g++
isobox pkg install make
isobox pkg install cmake
isobox pkg install autoconf
```

## Advanced Usage

### Finding Packages

Search Alpine packages online:
https://pkgs.alpinelinux.org/packages

Or use the command line (from host):
```bash
curl -s "https://pkgs.alpinelinux.org/packages?name=curl&branch=v3.19" | grep -o 'href="/packages/[^"]*"'
```

### Multiple Package Installation

```bash
isobox pkg install git
isobox pkg install curl
isobox pkg install vim
```

Or in a script:
```bash
for pkg in git curl vim python3; do
  isobox pkg install $pkg
done
```

### Package Dependencies

IPKG does **not** automatically resolve dependencies. You must install them manually.

Example for Python development:
```bash
isobox pkg install python3
isobox pkg install py3-pip
isobox pkg install python3-dev
```

### Checking Package Availability

Before installing, verify the package exists:

```bash
curl -I https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/<package>.apk
```

If you get a 200 response, the package exists.

### Package Versions

IPKG always installs the latest version from the configured Alpine repository. Version pinning is not currently supported.

To use a different Alpine version, you would need to modify the source code (see Technical Details).

## Environment Workflows

### Setting Up a Development Environment

```bash
cd ~/myproject
isobox init

isobox pkg install git
isobox pkg install vim
isobox pkg install make
isobox pkg install gcc

isobox enter
```

### Creating a Build Environment

```bash
cd ~/build-env
isobox init

isobox pkg install gcc
isobox pkg install make
isobox pkg install cmake
isobox pkg install git

isobox enter
```

### Setting Up a Scripting Environment

```bash
cd ~/scripts
isobox init

isobox pkg install bash
isobox pkg install curl
isobox pkg install jq
isobox pkg install python3

isobox enter
```

## Package Database

### Location

Each environment has its own database:
```
.isobox/var/lib/ipkg/installed.json
```

### Schema

```json
[
  {
    "name": "string",        // Package name
    "version": "string",     // Version from Alpine
    "description": "string", // Description
    "files": ["string"]      // Installed files (limited tracking)
  }
]
```

### Manual Inspection

```bash
cat .isobox/var/lib/ipkg/installed.json | jq
```

### Manual Editing

Not recommended, but you can manually edit the database:

```bash
vim .isobox/var/lib/ipkg/installed.json
```

## Limitations

### 1. No Dependency Resolution

You must manually install dependencies:

```bash
isobox pkg install python3
isobox pkg install py3-pip
```

### 2. No Version Pinning

Always installs the latest version from Alpine v3.19.

### 3. No Package Verification

No GPG signature checking (planned for future).

### 4. Limited File Tracking

The `files` array in the database is not fully populated during installation.

### 5. Basic Removal

`pkg remove` only removes the database entry, not the actual files (planned for future).

### 6. No Search

No built-in package search (use Alpine website).

### 7. Single Repository

Only supports one Alpine repository at a time.

## Troubleshooting

### Package Not Found

**Error:** `download failed: 404 Not Found`

**Cause:** Package doesn't exist in Alpine repository

**Solution:** 
1. Search for the correct package name: https://pkgs.alpinelinux.org/packages
2. Check spelling
3. Try alternative package names (e.g., `python3` not `python`)

### Download Fails

**Error:** `download failed: connection refused`

**Cause:** No internet connection or repository down

**Solution:**
1. Check internet: `ping 8.8.8.8`
2. Check repository: `curl https://dl-cdn.alpinelinux.org/alpine/`
3. Use cached or system binaries

### Extraction Fails

**Error:** `extract failed`

**Cause:** `tar` not available or corrupted download

**Solution:**
1. Ensure `tar` is available: `which tar`
2. Delete and retry: `rm /tmp/<package>.apk`
3. Install tar: `isobox pkg install tar`

### Permission Denied

**Error:** `permission denied`

**Cause:** Insufficient permissions to write to `.isobox/`

**Solution:**
1. Check ownership: `ls -la .isobox`
2. Fix permissions: `chmod -R u+w .isobox`

### Package Already Installed

**Message:** `Package X is already installed`

**Cause:** Package is in the database

**Solution:**
1. Remove first: `isobox pkg remove X`
2. Then reinstall: `isobox pkg install X`

## Technical Details

### APK File Format

Alpine packages (.apk) are tar.gz archives containing:
- `/usr/bin/`, `/usr/lib/`, etc. - Package files
- `.PKGINFO` - Package metadata
- `.trigger` - Installation triggers (optional)

IPKG extracts the entire archive to `.isobox/`.

### Download URL Structure

```
https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/<package>.apk
```

- `v3.19`: Alpine version
- `main`: Repository branch
- `x86_64`: Architecture
- `<package>.apk`: Package file

### Database Operations

All operations are atomic:
1. Read entire database
2. Modify in memory
3. Write back to disk

This ensures consistency even if interrupted.

### Fallback Mechanism

If download fails, IPKG attempts to use the system `apk` command:

```bash
apk add --root .isobox --initdb <package>
```

This requires `apk` to be installed on the host.

## Future Enhancements

Planned features:

1. **Dependency Resolution**: Automatic dependency installation
2. **GPG Verification**: Package signature checking
3. **Version Pinning**: Install specific versions
4. **Package Search**: Built-in search functionality
5. **Complete File Tracking**: Full file manifest
6. **Clean Removal**: Remove package files on `pkg remove`
7. **Multiple Repositories**: Support for additional repos
8. **Package Caching**: Local cache for faster reinstallation
9. **Upgrade Command**: `pkg upgrade` to update packages
10. **Lock File**: Prevent concurrent modifications

## Contributing

To add features to IPKG:

1. Edit `pkg/ipkg/manager.go`
2. Test thoroughly in multiple environments
3. Update this documentation
4. Submit a pull request

## Examples

### Full Python Environment

```bash
isobox init
isobox pkg install python3
isobox pkg install py3-pip
isobox pkg install python3-dev
isobox pkg install gcc
isobox pkg install musl-dev
isobox enter
python3 -m pip install requests
```

### Web Development Tools

```bash
isobox init
isobox pkg install nodejs
isobox pkg install npm
isobox pkg install git
isobox pkg install curl
isobox enter
```

### System Administration

```bash
isobox init
isobox pkg install bash
isobox pkg install coreutils
isobox pkg install findutils
isobox pkg install grep
isobox pkg install sed
isobox pkg install awk
isobox enter
```

## Best Practices

1. **Initialize first**: Always `isobox init` before installing packages
2. **Document dependencies**: Keep track of required packages
3. **One environment per project**: Don't share environments
4. **Check availability**: Verify packages exist before installing
5. **Manual dependencies**: Install dependencies explicitly
6. **Clean environments**: Remove unused packages
7. **Don't commit**: Add `.isobox/` to `.gitignore`

## Comparison

| Feature | IPKG | APK | apt | npm |
|---------|------|-----|-----|-----|
| Scope | Environment | System | System | Project |
| Dependencies | Manual | Auto | Auto | Auto |
| Database | JSON | SQLite | dpkg | package.json |
| Size | Tiny | Small | Large | Medium |
| Speed | Fast | Fast | Medium | Medium |

IPKG is designed for simplicity and isolation, not to replace system package managers.
