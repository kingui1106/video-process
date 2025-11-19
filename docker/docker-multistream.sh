#!/bin/bash

# Firescrew Multistream Docker Management Script
# Usage: ./docker-multistream.sh [command]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="$PROJECT_DIR/docker-compose.yml"
CONFIG_FILE="$PROJECT_DIR/config.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Check if docker-compose is installed
check_docker_compose() {
    if ! command -v docker-compose &> /dev/null; then
        error "docker-compose is not installed. Please install it first."
    fi
}

# Check if config file exists
check_config() {
    if [ ! -f "$CONFIG_FILE" ]; then
        warn "Config file not found: $CONFIG_FILE"
        warn "Creating a sample config file..."
        cat > "$CONFIG_FILE" << 'EOF'
{
  "webPort": ":8081",
  "cameras": [
    {
      "id": "camera1",
      "name": "示例摄像头",
      "rtspUrl": "rtsp://192.168.1.100:554/stream",
      "roi": [],
      "drawElements": [],
      "enabled": true
    }
  ]
}
EOF
        info "Sample config created. Please edit $CONFIG_FILE with your camera settings."
    fi
}

# Build the Docker image
build() {
    info "Building Docker image..."
    cd "$PROJECT_DIR"
    docker-compose build
    info "Build completed successfully!"
}

# Start the service
start() {
    info "Starting Firescrew Multistream..."
    check_config
    cd "$PROJECT_DIR"
    docker-compose up -d
    info "Service started successfully!"
    info "Web interface: http://localhost:8081/config"
}

# Stop the service
stop() {
    info "Stopping Firescrew Multistream..."
    cd "$PROJECT_DIR"
    docker-compose down
    info "Service stopped successfully!"
}

# Restart the service
restart() {
    info "Restarting Firescrew Multistream..."
    stop
    start
}

# View logs
logs() {
    cd "$PROJECT_DIR"
    docker-compose logs -f
}

# Show status
status() {
    cd "$PROJECT_DIR"
    docker-compose ps
}

# Clean up (remove containers, images, volumes)
clean() {
    warn "This will remove all containers, images, and volumes. Are you sure? (y/N)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        info "Cleaning up..."
        cd "$PROJECT_DIR"
        docker-compose down -v --rmi all
        info "Cleanup completed!"
    else
        info "Cleanup cancelled."
    fi
}

# Show help
show_help() {
    cat << EOF
Firescrew Multistream Docker Management Script

Usage: $0 [command]

Commands:
    build       Build the Docker image
    start       Start the service
    stop        Stop the service
    restart     Restart the service
    logs        View service logs (follow mode)
    status      Show service status
    clean       Remove containers, images, and volumes
    help        Show this help message

Examples:
    $0 build
    $0 start
    $0 logs
    $0 stop

EOF
}

# Main script
main() {
    check_docker_compose

    case "${1:-}" in
        build)
            build
            ;;
        start)
            start
            ;;
        stop)
            stop
            ;;
        restart)
            restart
            ;;
        logs)
            logs
            ;;
        status)
            status
            ;;
        clean)
            clean
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            show_help
            exit 1
            ;;
    esac
}

main "$@"

