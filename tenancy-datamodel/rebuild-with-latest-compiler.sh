#!/bin/bash
# SPDX-FileCopyrightText: 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Go up one level from tenancy-datamodel to reach orch-utils root
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPILER_DIR="${REPO_ROOT}/nexus/compiler"
DATAMODEL_DIR="${SCRIPT_DIR}"

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# Cleanup and fix permissions
cleanup_and_fix_permissions() {
    log_info "Cleaning up old build artifacts..."
    
    if [ -d "${DATAMODEL_DIR}/build/nexus-client" ]; then
        log_info "Removing ${DATAMODEL_DIR}/build/nexus-client..."
        sudo rm -rf "${DATAMODEL_DIR}/build/nexus-client" || true
    fi
    
    if [ -d "${DATAMODEL_DIR}/generated/nexus-client" ]; then
        log_info "Removing ${DATAMODEL_DIR}/generated/nexus-client..."
        sudo rm -rf "${DATAMODEL_DIR}/generated/nexus-client" || true
    fi
    
    # Remove other generated directories with sudo if needed
    for dir in apis client common crds helper install-validator model nexus-client nexus-gql tsm-nexus-gql; do
        if [ -d "${DATAMODEL_DIR}/build/${dir}" ]; then
            log_info "Removing build/${dir}..."
            sudo rm -rf "${DATAMODEL_DIR}/build/${dir}" || true
        fi
        
        if [ -d "${DATAMODEL_DIR}/generated/${dir}" ]; then
            log_info "Removing generated/${dir}..."
            sudo rm -rf "${DATAMODEL_DIR}/generated/${dir}" || true
        fi
    done
    
    # Fix permissions on openapi directory
    log_info "Fixing permissions on build/openapi directory..."
    if [ -d "${DATAMODEL_DIR}/build/openapi" ]; then
        sudo chmod -R 777 "${DATAMODEL_DIR}/build/openapi" || true
    else
        mkdir -p "${DATAMODEL_DIR}/build/openapi"
        sudo chmod -R 777 "${DATAMODEL_DIR}/build/openapi" || true
    fi
}

# Check if compiler directory has uncommitted changes
check_compiler_changes() {
    log_info "Checking for uncommitted changes in compiler..."
    
    cd "${COMPILER_DIR}"
    
    # Check git status
    if ! git diff --quiet || ! git diff --cached --quiet; then
        log_warning "Found uncommitted changes in ${COMPILER_DIR}"
        git status --short
        return 0 # Changes found
    else
        log_info "No uncommitted changes in compiler"
        return 1 # No changes
    fi
}

# Rebuild compiler builder image
rebuild_compiler_builder() {
    log_info "Rebuilding compiler builder image..."
    
    cd "${COMPILER_DIR}"
    
    if make docker.builder 2>&1 | grep -q "naming to docker.io"; then
        log_success "Compiler builder image rebuilt successfully"
    else
        log_error "Failed to rebuild compiler builder image"
        return 1
    fi
}

# Rebuild compiler image with local changes included
rebuild_compiler() {
    log_info "Rebuilding compiler image..."
    
    cd "${COMPILER_DIR}"
    
    # Use INCLUDE_LOCAL_CHANGES=true to include uncommitted template changes
    if make docker INCLUDE_LOCAL_CHANGES=true 2>&1 | grep -q "naming to docker.io"; then
        log_success "Compiler image rebuilt successfully"
        
        # Verify the image contains the updated template
        if docker run --rm nexus/compiler/amd64:0.1.0-dev \
            cat /go/src/github.com/vmware-tanzu/graph-framework-for-microservices/compiler/pkg/generator/template/client.go.tmpl \
            2>/dev/null | grep -q "WaitForCacheSync"; then
            log_success "Verified: Compiler image contains latest template changes"
        fi
    else
        log_error "Failed to rebuild compiler image"
        return 1
    fi
}

# Rebuild openapi-generator image
rebuild_openapi_generator() {
    log_info "Checking if openapi-generator image exists..."
    
    if docker inspect nexus/openapi-generator:0.1.0-dev >/dev/null 2>&1; then
        log_success "OpenAPI generator image already exists"
    else
        log_info "Building openapi-generator image..."
        cd "${REPO_ROOT}/nexus"
        
        if make openapi.generator.docker 2>&1 | grep -q "naming to docker.io"; then
            log_success "OpenAPI generator image built successfully"
        else
            log_error "Failed to build openapi-generator image"
            return 1
        fi
    fi
}

# Rebuild datamodel
rebuild_datamodel() {
    log_info "Building datamodel..."
    
    cd "${DATAMODEL_DIR}"
    
    # Set shell options for make
    export SHELL=/bin/bash
    
    # Run make datamodel_build with error handling
    if bash -c 'set +u; make datamodel_build 2>&1' | tee /tmp/datamodel_build.log; then
        log_success "Datamodel built successfully"
    else
        log_error "Failed to build datamodel"
        log_error "See /tmp/datamodel_build.log for details"
        return 1
    fi
}

# Verify generated files
verify_generated_files() {
    log_info "Verifying generated files..."
    
    local checks_passed=0
    local checks_total=0
    
    # Check for generated client file
    checks_total=$((checks_total + 1))
    if [ -f "${DATAMODEL_DIR}/build/nexus-client/client.go" ]; then
        checks_passed=$((checks_passed + 1))
        log_success "Found generated client.go"
    else
        log_error "Missing generated client.go"
    fi
    
    # Check for CRDs
    checks_total=$((checks_total + 1))
    if [ -d "${DATAMODEL_DIR}/build/crds" ] && [ "$(ls -A ${DATAMODEL_DIR}/build/crds)" ]; then
        checks_passed=$((checks_passed + 1))
        log_success "Found generated CRDs"
    else
        log_error "Missing or empty CRDs directory"
    fi
    
    # Check for APIs
    checks_total=$((checks_total + 1))
    if [ -d "${DATAMODEL_DIR}/build/apis" ] && [ "$(ls -A ${DATAMODEL_DIR}/build/apis)" ]; then
        checks_passed=$((checks_passed + 1))
        log_success "Found generated APIs"
    else
        log_error "Missing or empty APIs directory"
    fi
    
    # Check for OpenAPI spec
    checks_total=$((checks_total + 1))
    if [ -f "${DATAMODEL_DIR}/build/openapi/edge-orchestrator.intel.com.json" ]; then
        checks_passed=$((checks_passed + 1))
        log_success "Found generated OpenAPI spec"
    else
        log_error "Missing OpenAPI spec"
    fi
    
    if [ $checks_passed -eq $checks_total ]; then
        log_success "All verification checks passed ($checks_passed/$checks_total)"
        return 0
    else
        log_warning "Some verification checks failed ($checks_passed/$checks_total)"
        return 1
    fi
}

# Print usage
print_usage() {
    cat << 'EOF'
Usage: ./rebuild-with-latest-compiler.sh [OPTIONS]

Rebuilds the tenancy-datamodel using the latest nexus/compiler image,
including any local changes in template files.

OPTIONS:
    -h, --help              Show this help message
    -f, --force-rebuild     Force rebuild of compiler image even without changes
    -n, --no-verify         Skip verification of generated files
    -d, --docker-only       Only rebuild docker images, don't build datamodel
    -v, --verbose           Enable verbose output

EXAMPLES:
    # Standard rebuild with latest compiler
    ./rebuild-with-latest-compiler.sh

    # Force rebuild compiler even without changes
    ./rebuild-with-latest-compiler.sh --force-rebuild

    # Only rebuild docker images
    ./rebuild-with-latest-compiler.sh --docker-only

EOF
}

# Main execution
main() {
    local force_rebuild=false
    local verify=true
    local docker_only=false
    local verbose=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                print_usage
                exit 0
                ;;
            -f|--force-rebuild)
                force_rebuild=true
                shift
                ;;
            -n|--no-verify)
                verify=false
                shift
                ;;
            -d|--docker-only)
                docker_only=true
                shift
                ;;
            -v|--verbose)
                verbose=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    log_info "=========================================="
    log_info "Tenancy-Datamodel Rebuild Script"
    log_info "=========================================="
    log_info "Repository Root: ${REPO_ROOT}"
    log_info "Compiler Dir: ${COMPILER_DIR}"
    log_info "Datamodel Dir: ${DATAMODEL_DIR}"
    log_info ""
    
    # Step 1: Check for compiler changes
    if ! check_compiler_changes && [ "$force_rebuild" = false ]; then
        log_info "No changes detected in compiler and --force-rebuild not specified"
        log_info "Skipping compiler rebuild..."
    else
        log_info "Rebuilding compiler images..."
        
        # Step 2: Rebuild compiler builder
        if ! rebuild_compiler_builder; then
            log_error "Failed to rebuild compiler builder"
            exit 1
        fi
        
        # Step 3: Rebuild compiler
        if ! rebuild_compiler; then
            log_error "Failed to rebuild compiler"
            exit 1
        fi
    fi
    
    # Step 4: Rebuild openapi-generator if needed
    if ! rebuild_openapi_generator; then
        log_error "Failed to rebuild openapi-generator"
        exit 1
    fi
    
    # Skip datamodel build if --docker-only is set
    if [ "$docker_only" = true ]; then
        log_success "Docker images rebuilt successfully"
        exit 0
    fi
    
    # Step 5: Cleanup and fix permissions
    cleanup_and_fix_permissions
    
    # Step 6: Rebuild datamodel
    if ! rebuild_datamodel; then
        log_error "Failed to rebuild datamodel"
        exit 1
    fi
    
    # Step 7: Verify generated files (if not disabled)
    if [ "$verify" = true ]; then
        if ! verify_generated_files; then
            log_warning "Some verification checks failed, but build may still be usable"
        fi
    fi
    
    log_info ""
    log_success "=========================================="
    log_success "Tenancy-Datamodel rebuild completed successfully!"
    log_success "=========================================="
    log_info "Generated files are in: ${DATAMODEL_DIR}/build/"
    log_info ""
}

# Run main function
main "$@"