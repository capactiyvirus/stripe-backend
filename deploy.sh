#!/bin/bash

# Docker Deployment Script for Stripe Backend
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is installed and running
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi

    if ! docker info &> /dev/null; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
}

# Create necessary directories
create_directories() {
    print_status "Creating necessary directories..."
    mkdir -p db/init
    mkdir -p logs
    mkdir -p data/postgres
    mkdir -p data/redis
}

# Setup environment file
setup_env() {
    if [ ! -f .env ]; then
        print_status "Creating .env file from template..."
        cp .env.docker .env
        print_warning "Please edit .env file with your actual Stripe keys and other configuration"
        print_warning "The application will not work properly without valid Stripe keys"
    else
        print_status ".env file already exists"
    fi
}

# Build and start services
start_services() {
    print_status "Building and starting services..."
    
    # Pull latest images
    docker-compose pull
    
    # Build the application
    docker-compose build --no-cache stripe-backend
    
    # Start services
    docker-compose up -d
    
    print_success "Services started successfully!"
}

# Check service health
check_health() {
    print_status "Checking service health..."
    
    # Wait for services to be ready
    sleep 10
    
    # Check PostgreSQL
    if docker-compose exec -T postgres pg_isready -U stripe_user -d stripe_payments > /dev/null 2>&1; then
        print_success "PostgreSQL is ready"
    else
        print_error "PostgreSQL is not ready"
    fi
    
    # Check application
    if curl -f http://localhost:8080/health > /dev/null 2>&1; then
        print_success "Application is ready"
    else
        print_error "Application is not ready"
    fi
}

# Show service status
show_status() {
    print_status "Service Status:"
    docker-compose ps
    
    echo ""
    print_status "Service URLs:"
    echo "  - API: http://localhost:8080"
    echo "  - Health Check: http://localhost:8080/health"
    echo "  - pgAdmin: http://localhost:5050 (admin@stripe.local / admin123)"
    echo "  - PostgreSQL: localhost:5432 (stripe_user / stripe_password123)"
    echo "  - Redis: localhost:6379"
}

# Show logs
show_logs() {
    if [ "$1" == "follow" ]; then
        docker-compose logs -f
    else
        docker-compose logs --tail=50
    fi
}

# Stop services
stop_services() {
    print_status "Stopping services..."
    docker-compose down
    print_success "Services stopped"
}

# Clean up (remove containers, networks, volumes)
cleanup() {
    print_warning "This will remove all containers, networks, and volumes!"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Cleaning up..."
        docker-compose down -v --remove-orphans
        docker system prune -f
        print_success "Cleanup completed"
    else
        print_status "Cleanup cancelled"
    fi
}

# Main script logic
case "$1" in
    "start"|"up")
        check_docker
        create_directories
        setup_env
        start_services
        check_health
        show_status
        ;;
    "stop"|"down")
        stop_services
        ;;
    "restart")
        stop_services
        sleep 2
        check_docker
        start_services
        check_health
        show_status
        ;;
    "status")
        show_status
        ;;
    "logs")
        show_logs "$2"
        ;;
    "health")
        check_health
        ;;
    "cleanup")
        cleanup
        ;;
    "build")
        print_status "Building application..."
        docker-compose build --no-cache stripe-backend
        print_success "Build completed"
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|logs|health|cleanup|build}"
        echo ""
        echo "Commands:"
        echo "  start    - Start all services"
        echo "  stop     - Stop all services"
        echo "  restart  - Restart all services"
        echo "  status   - Show service status and URLs"
        echo "  logs     - Show service logs (add 'follow' to tail logs)"
        echo "  health   - Check service health"
        echo "  cleanup  - Remove all containers, networks, and volumes"
        echo "  build    - Rebuild the application container"
        echo ""
        echo "Examples:"
        echo "  ./deploy.sh start"
        echo "  ./deploy.sh logs follow"
        echo "  ./deploy.sh stop"
        exit 1
        ;;
esac