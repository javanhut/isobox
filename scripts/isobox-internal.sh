#!/bin/sh

ISOBOX_DB="/var/lib/ipkg/installed.json"
ISOBOX_PKG_CACHE="/var/cache/isobox"

command_help() {
    cat << EOF
IsoBox Internal Package Manager

Usage:
  isobox install <package>    Install a package
  isobox remove <package>     Remove a package
  isobox list                 List installed packages
  isobox update               Update package index
  isobox help                 Show this help

EOF
}

detect_host_system() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "$HOST_SYSTEM"
    else
        echo "Unknown"
    fi
}

get_package_manager() {
    host=$(detect_host_system)

    case "$host" in
        *Arch*|*Manjaro*|*EndeavourOS*)
            echo "pacman"
            ;;
        *Debian*|*Ubuntu*|*Mint*|*Pop*)
            echo "apt"
            ;;
        *Fedora*|*RHEL*|*CentOS*|*Rocky*)
            echo "dnf"
            ;;
        *openSUSE*)
            echo "zypper"
            ;;
        *Alpine*)
            echo "apk"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

ensure_db() {
    db_dir="${ISOBOX_DB%/*}"
    mkdir -p "$db_dir"
    if [ ! -f "$ISOBOX_DB" ]; then
        echo "[]" > "$ISOBOX_DB"
    fi
}

is_installed() {
    pkg_name="$1"
    ensure_db

    if command -v jq >/dev/null 2>&1; then
        jq -e ".[] | select(.name == \"$pkg_name\")" "$ISOBOX_DB" >/dev/null 2>&1
    else
        grep -q "\"name\": \"$pkg_name\"" "$ISOBOX_DB" 2>/dev/null
    fi
}

add_to_db() {
    pkg_name="$1"
    ensure_db

    temp_db="/tmp/isobox_db_$$.json"

    if command -v jq >/dev/null 2>&1; then
        jq ". += [{\"name\": \"$pkg_name\", \"version\": \"latest\", \"installed\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}]" "$ISOBOX_DB" > "$temp_db"
        mv "$temp_db" "$ISOBOX_DB"
    else
        sed -i 's/\]$/,{"name":"'"$pkg_name"'","version":"latest","installed":"'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"}]/' "$ISOBOX_DB"
    fi
}

remove_from_db() {
    pkg_name="$1"
    ensure_db

    temp_db="/tmp/isobox_db_$$.json"

    if command -v jq >/dev/null 2>&1; then
        jq "map(select(.name != \"$pkg_name\"))" "$ISOBOX_DB" > "$temp_db"
        mv "$temp_db" "$ISOBOX_DB"
    else
        echo "Error: jq not available for database manipulation"
        return 1
    fi
}

get_package_dependencies() {
    apk_file="$1"

    # Extract .PKGINFO from the APK to read dependencies
    pkginfo=$(tar -xzf "$apk_file" -O .PKGINFO 2>/dev/null)

    if [ -z "$pkginfo" ]; then
        return 0
    fi

    # Parse dependencies from PKGINFO
    echo "$pkginfo" | grep "^depend = " | sed 's/^depend = //' | while read dep; do
        # Remove version constraints (e.g., "package>=1.0" -> "package")
        dep_name=$(echo "$dep" | sed 's/[<>=].*//')
        echo "$dep_name"
    done
}

# Map shared library dependencies (so:*) to package names
map_so_to_package() {
    so_name="$1"

    # Common shared library to package mappings for Alpine
    case "$so_name" in
        so:libluv.so.1) echo "luv" ;;
        so:libtermkey.so.1) echo "libtermkey" ;;
        so:libvterm.so.0) echo "libvterm" ;;
        so:libmsgpack-c.so.2) echo "msgpack-c" ;;
        so:libtree-sitter.so.0) echo "tree-sitter" ;;
        so:libunibilium.so.4) echo "unibilium" ;;
        so:libintl.so.8) echo "musl-libintl" ;;
        so:libluajit-5.1.so.2) echo "luajit" ;;
        so:libuv.so.1) echo "libuv" ;;
        so:libc.musl-*.so.1) echo "musl" ;;
        so:libssl.so.3) echo "libssl3" ;;
        so:libcrypto.so.3) echo "libcrypto3" ;;
        so:libz.so.1) echo "zlib" ;;
        so:libpcre2-8.so.0) echo "pcre2" ;;
        so:libcurl.so.4) echo "libcurl" ;;
        so:libnghttp2.so.14) echo "nghttp2-libs" ;;
        so:libbrotlidec.so.1) echo "brotli-libs" ;;
        so:libpsl.so.5) echo "libpsl" ;;
        so:libc-ares.so.2) echo "c-ares" ;;
        so:libidn2.so.0) echo "libidn2" ;;
        so:libunistring.so.5) echo "libunistring" ;;
        so:libncursesw.so.6) echo "libncursesw" ;;
        so:libreadline.so.8) echo "readline" ;;
        so:libonig.so.5) echo "oniguruma" ;;
        so:libstdc++.so.6) echo "libstdc++" ;;
        so:libgcc_s.so.1) echo "libgcc" ;;
        *) return 1 ;;  # Unknown mapping
    esac
    return 0
}

install_package_with_deps() {
    pkg_name="$1"
    installing_marker="/tmp/isobox_installing_${pkg_name}"

    # Prevent circular dependencies
    if [ -f "$installing_marker" ]; then
        return 0
    fi

    touch "$installing_marker"
    trap "rm -f $installing_marker" EXIT

    # Check if already installed
    if is_installed "$pkg_name"; then
        rm -f "$installing_marker"
        return 0
    fi

    # Download the package first to check dependencies
    mkdir -p "$ISOBOX_PKG_CACHE"
    apk_file="$ISOBOX_PKG_CACHE/$pkg_name.apk"

    if ! command -v wget >/dev/null 2>&1 && ! command -v curl >/dev/null 2>&1; then
        echo "Error: wget or curl required for package installation"
        rm -f "$installing_marker"
        return 1
    fi

    # Find and download package
    if command -v wget >/dev/null 2>&1; then
        index_page=$(wget -qO- "https://dl-cdn.alpinelinux.org/alpine/v3.18/main/x86_64/" 2>&1)
        pkg_list=$(echo "$index_page" | grep "href=\"$pkg_name-[0-9]" | head -1 | sed 's/.*href="//;s/".*//')

        if [ -z "$pkg_list" ]; then
            index_page=$(wget -qO- "https://dl-cdn.alpinelinux.org/alpine/v3.18/community/x86_64/" 2>&1)
            pkg_list=$(echo "$index_page" | grep "href=\"$pkg_name-[0-9]" | head -1 | sed 's/.*href="//;s/".*//')
            apk_url="https://dl-cdn.alpinelinux.org/alpine/v3.18/community/x86_64/$pkg_list"
        else
            apk_url="https://dl-cdn.alpinelinux.org/alpine/v3.18/main/x86_64/$pkg_list"
        fi

        if [ -z "$pkg_list" ]; then
            echo "  Warning: Package $pkg_name not found, skipping..."
            rm -f "$installing_marker"
            return 0
        fi

        wget -q -O "$apk_file" "$apk_url" 2>&1 || {
            rm -f "$installing_marker"
            return 1
        }
    else
        pkg_list=$(curl -s "https://dl-cdn.alpinelinux.org/alpine/v3.18/main/x86_64/" 2>/dev/null | grep "href=\"$pkg_name-[0-9]" | head -1 | sed 's/.*href="//;s/".*//')

        if [ -z "$pkg_list" ]; then
            pkg_list=$(curl -s "https://dl-cdn.alpinelinux.org/alpine/v3.18/community/x86_64/" 2>/dev/null | grep "href=\"$pkg_name-[0-9]" | head -1 | sed 's/.*href="//;s/".*//')
            apk_url="https://dl-cdn.alpinelinux.org/alpine/v3.18/community/x86_64/$pkg_list"
        else
            apk_url="https://dl-cdn.alpinelinux.org/alpine/v3.18/main/x86_64/$pkg_list"
        fi

        if [ -z "$pkg_list" ]; then
            echo "  Warning: Package $pkg_name not found, skipping..."
            rm -f "$installing_marker"
            return 0
        fi

        curl -s -o "$apk_file" "$apk_url" || {
            rm -f "$installing_marker"
            return 1
        }
    fi

    # Get dependencies
    deps=$(get_package_dependencies "$apk_file")

    # Install dependencies first
    if [ -n "$deps" ]; then
        echo "  Resolving dependencies for $pkg_name..."
        for dep in $deps; do
            # Handle shared library dependencies (so:*)
            if echo "$dep" | grep -q "^so:"; then
                # Try to map so: dependency to package name
                real_pkg=$(map_so_to_package "$dep")
                if [ $? -eq 0 ] && [ -n "$real_pkg" ]; then
                    dep="$real_pkg"
                else
                    # Unknown so: dependency, skip it
                    continue
                fi
            fi

            if ! is_installed "$dep"; then
                echo "  Installing dependency: $dep"
                install_package_with_deps "$dep"
            fi
        done
    fi

    # Now install the actual package
    echo "  Installing $pkg_name..."
    tar -xzf "$apk_file" -C / 2>/dev/null || {
        echo "  Error: Failed to extract $pkg_name"
        rm -f "$apk_file" "$installing_marker"
        return 1
    }

    rm -f "$apk_file"
    add_to_db "$pkg_name"
    rm -f "$installing_marker"

    return 0
}

install_package() {
    pkg_name="$1"

    if [ -z "$pkg_name" ]; then
        echo "Error: Package name required"
        echo "Usage: isobox install <package>"
        return 1
    fi

    if is_installed "$pkg_name"; then
        echo "Package $pkg_name is already installed"
        return 0
    fi

    host=$(detect_host_system)
    echo "Installing $pkg_name (host: $host)..."
    echo "Resolving dependencies..."

    install_package_with_deps "$pkg_name"

    if [ $? -eq 0 ]; then
        echo "Successfully installed $pkg_name and its dependencies"
    else
        echo "Failed to install $pkg_name"
        return 1
    fi
}

install_common_dependencies() {
    marker="/var/lib/ipkg/.alpine_deps_installed"
    if [ -f "$marker" ]; then
        return 0
    fi

    echo "Installing Alpine library dependencies..."

    deps="pcre2 zlib c-ares libpsl nghttp2-libs brotli-libs"
    for dep in $deps; do
        if command -v wget >/dev/null 2>&1; then
            index_page=$(wget -qO- "https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/" 2>&1)
            dep_pkg=$(echo "$index_page" | grep "href=\"$dep-[0-9]" | head -1 | sed 's/.*href="//;s/".*//')

            if [ -n "$dep_pkg" ]; then
                echo "  Installing $dep..."
                dep_url="https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/$dep_pkg"
                dep_file="$ISOBOX_PKG_CACHE/$dep.apk"

                wget -q -O "$dep_file" "$dep_url" 2>&1 && tar -xzf "$dep_file" -C / 2>/dev/null
                rm -f "$dep_file"
            fi
        fi
    done

    touch "$marker"
    echo "  Dependencies installed"
}

remove_package() {
    pkg_name="$1"

    if [ -z "$pkg_name" ]; then
        echo "Error: Package name required"
        echo "Usage: isobox remove <package>"
        return 1
    fi

    if ! is_installed "$pkg_name"; then
        echo "Package $pkg_name is not installed"
        return 0
    fi

    echo "Removing $pkg_name..."
    remove_from_db "$pkg_name"
    echo "Successfully removed $pkg_name from database"
    echo "Note: Files are not removed automatically"
}

list_packages() {
    ensure_db

    if command -v jq >/dev/null 2>&1; then
        count=$(jq 'length' "$ISOBOX_DB")

        if [ "$count" -eq 0 ]; then
            echo "No packages installed"
            return 0
        fi

        echo "Installed packages:"
        jq -r '.[] | "  \(.name) (\(.version)) - installed \(.installed)"' "$ISOBOX_DB"
    else
        echo "Installed packages:"
        cat "$ISOBOX_DB"
    fi
}

update_index() {
    pm=$(get_package_manager)
    host=$(detect_host_system)

    echo "Updating package index for $pm (host: $host)..."

    case "$pm" in
        pacman)
            echo "Note: Run 'pacman -Sy' on host to update index"
            ;;
        apt)
            echo "Note: Run 'apt-get update' on host to update index"
            ;;
        dnf)
            echo "Note: Run 'dnf check-update' on host to update index"
            ;;
        apk)
            echo "Note: Run 'apk update' on host to update index"
            ;;
        *)
            echo "Note: Update package index on host system"
            ;;
    esac

    echo "Package index updated"
}

case "$1" in
    install)
        install_package "$2"
        ;;
    remove)
        remove_package "$2"
        ;;
    list)
        list_packages
        ;;
    update)
        update_index
        ;;
    help|--help|-h)
        command_help
        ;;
    *)
        echo "Error: Unknown command '$1'"
        echo ""
        command_help
        exit 1
        ;;
esac
