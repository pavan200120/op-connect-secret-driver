#!/bin/bash
set -e

# Default values
DEFAULT_HOST="http://localhost:17450"
DEFAULT_TOKEN=""

# Help function
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Install and configure the 1Password Connect Secret Driver for Docker"
    echo
    echo "Options:"
    echo "  --help           Show this help message"
    echo "  --host HOST     1Password Connect server URL (default: $DEFAULT_HOST)"
    echo "  --token TOKEN   1Password Connect Token (required)"
    echo
    echo "Environment variables:"
    echo "  OP_CONNECT_HOST  Alternative to --host"
    echo "  OP_CONNECT_TOKEN Alternative to --token"
    echo
    echo "Examples:"
    echo "  $0 --token \"your-token\""
    echo "  $0 --host \"http://custom-host:17450\" --token \"your-token\""
    echo "  OP_CONNECT_TOKEN=\"your-token\" $0"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --help|-h)
            show_help
            exit 0
            ;;
        --host)
            OP_CONNECT_HOST="$2"
            shift 2
            ;;
        --token)
            OP_CONNECT_TOKEN="$2"
            shift 2
            ;;
        *)
            echo "Error: Unknown parameter: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Use environment variables or defaults
OP_CONNECT_HOST=${OP_CONNECT_HOST:-$DEFAULT_HOST}
OP_CONNECT_TOKEN=${OP_CONNECT_TOKEN:-$DEFAULT_TOKEN}

# Validate required parameters
if [ -z "$OP_CONNECT_TOKEN" ]; then
    echo "Error: OP_CONNECT_TOKEN is required. Set it via environment variable or --token argument"
    exit 1
fi

# cleanup
docker compose down > /dev/null 2>&1 || true
docker plugin disable op-connect-secret-driver:latest > /dev/null 2>&1 || true
docker plugin remove op-connect-secret-driver:latest > /dev/null 2>&1 || true

# build
docker compose build op-connect-secret-driver
docker compose up -d op-connect-secret-driver
docker compose cp op-connect-secret-driver:/usr/bin/op-connect-secret-driver plugin/rootfs/usr/bin/op-connect-secret-driver
docker compose stop op-connect-secret-driver && docker compose rm -f op-connect-secret-driver

# create
docker plugin create op-connect-secret-driver plugin

# set the connection details
docker plugin set op-connect-secret-driver:latest OP_CONNECT_HOST="$OP_CONNECT_HOST"
docker plugin set op-connect-secret-driver:latest OP_CONNECT_TOKEN="$OP_CONNECT_TOKEN"

# enable
docker compose up -d op-connect-api
docker plugin enable op-connect-secret-driver:latest