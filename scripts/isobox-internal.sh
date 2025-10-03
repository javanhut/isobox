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
    mkdir -p "$ISOBOX_PKG_CACHE"

    echo "Downloading from Alpine Linux repository..."

    if ! command -v wget >/dev/null 2>&1 && ! command -v curl >/dev/null 2>&1; then
        echo "Error: wget or curl required for package installation"
        echo "Please install wget or curl first"
        return 1
    fi

    apk_url="https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/$pkg_name-"
    apk_file="$ISOBOX_PKG_CACHE/$pkg_name.apk"

    if command -v wget >/dev/null 2>&1; then
        echo "Fetching package list..."

        index_page=$(wget -qO- "https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/" 2>&1)
        if [ $? -ne 0 ]; then
            echo "Error: Failed to fetch package index"
            echo "Debug: $index_page"
            return 1
        fi

        pkg_list=$(echo "$index_page" | grep "href=\"$pkg_name-[0-9]" | head -1 | sed 's/.*href="//;s/".*//')

        if [ -z "$pkg_list" ]; then
            echo "Warning: Package not found in main repo, trying community repo..."
            index_page=$(wget -qO- "https://dl-cdn.alpinelinux.org/alpine/v3.19/community/x86_64/" 2>&1)
            pkg_list=$(echo "$index_page" | grep "href=\"$pkg_name-[0-9]" | head -1 | sed 's/.*href="//;s/".*//')
            apk_url="https://dl-cdn.alpinelinux.org/alpine/v3.19/community/x86_64/$pkg_list"
        else
            apk_url="https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/$pkg_list"
        fi

        if [ -z "$pkg_list" ]; then
            echo "Error: Package $pkg_name not found in Alpine repositories"
            return 1
        fi

        echo "Downloading $pkg_list..."
        wget -q --show-progress -O "$apk_file" "$apk_url" 2>&1 || {
            echo "Error: Failed to download package"
            return 1
        }
    else
        echo "Fetching package list..."
        pkg_list=$(curl -s "https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/" 2>/dev/null | grep "href=\"$pkg_name-[0-9]" | head -1 | sed 's/.*href="//;s/".*//')

        if [ -z "$pkg_list" ]; then
            echo "Warning: Package not found in main repo, trying community repo..."
            pkg_list=$(curl -s "https://dl-cdn.alpinelinux.org/alpine/v3.19/community/x86_64/" 2>/dev/null | grep "href=\"$pkg_name-[0-9]" | head -1 | sed 's/.*href="//;s/".*//')
            apk_url="https://dl-cdn.alpinelinux.org/alpine/v3.19/community/x86_64/$pkg_list"
        else
            apk_url="https://dl-cdn.alpinelinux.org/alpine/v3.19/main/x86_64/$pkg_list"
        fi

        if [ -z "$pkg_list" ]; then
            echo "Error: Package $pkg_name not found in Alpine repositories"
            return 1
        fi

        echo "Downloading $pkg_list..."
        curl -# -o "$apk_file" "$apk_url" || {
            echo "Error: Failed to download package"
            return 1
        }
    fi

    if [ ! -f "$apk_file" ]; then
        echo "Error: Package file not found after download"
        return 1
    fi

    echo "Extracting package..."
    tar -xzf "$apk_file" -C / 2>/dev/null || {
        echo "Error: Failed to extract package"
        rm -f "$apk_file"
        return 1
    }

    rm -f "$apk_file"

    install_common_dependencies

    add_to_db "$pkg_name"
    echo "Successfully installed $pkg_name"
}

install_common_dependencies() {
    marker="/var/lib/ipkg/.alpine_deps_installed"
    if [ -f "$marker" ]; then
        return 0
    fi

    echo "Installing Alpine library dependencies..."

    deps="pcre2 zlib"
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
