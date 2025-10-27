#!/bin/bash

# Simple template script for use with blender.sh
# This script takes exactly one argument: the host
# Usage: ./template.sh <host>

# Get the host argument
HOST="$1"

# Validate that host is provided
if [[ -z "$HOST" ]]; then
    echo "Error: Host argument is required"
    echo "Usage: $0 <host>"
    exit 1
fi

# Function to log messages with timestamp
log() {
    echo "[$(date '+%H:%M:%S')] $*"
}

# Function to report success
success() {
    echo "✓ $*"
}

# Function to report failure
failure() {
    echo "✗ $*"
}

# Main script logic starts here
log "Processing host: $HOST"

# Example operation 1: Ping test
log "Testing connectivity..."
if ping -c 1 -W 3 "$HOST" >/dev/null 2>&1; then
    success "Host $HOST is reachable"
    PING_SUCCESS=true
else
    failure "Host $HOST is not reachable"
    PING_SUCCESS=false
fi

# Example operation 2: Port check (only if ping succeeded)
if [[ "$PING_SUCCESS" == "true" ]]; then
    log "Checking SSH port..."
    if timeout 5 bash -c "</dev/tcp/$HOST/22" 2>/dev/null; then
        success "SSH port 22 is open on $HOST"
        SSH_AVAILABLE=true
    else
        failure "SSH port 22 is not accessible on $HOST"
        SSH_AVAILABLE=false
    fi
fi

# Example operation 3: Additional checks based on previous results
if [[ "$SSH_AVAILABLE" == "true" ]]; then
    log "Attempting SSH connection..."
    
    # Example: Get hostname via SSH
    if REMOTE_HOSTNAME=$(timeout 10 ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no -o BatchMode=yes "$HOST" 'hostname' 2>/dev/null); then
        success "SSH successful - Remote hostname: $REMOTE_HOSTNAME"
        EXIT_CODE=0
    else
        failure "SSH connection failed to $HOST"
        EXIT_CODE=1
    fi
else
    # Set exit code based on ping result
    if [[ "$PING_SUCCESS" == "true" ]]; then
        EXIT_CODE=1  # Reachable but SSH not available
    else
        EXIT_CODE=2  # Not reachable at all
    fi
fi

# Summary
log "Summary for $HOST:"
echo "  Ping: $([ "$PING_SUCCESS" == "true" ] && echo "✓" || echo "✗")"
echo "  SSH:  $([ "$SSH_AVAILABLE" == "true" ] && echo "✓" || echo "✗")"

log "Finished processing $HOST (exit code: $EXIT_CODE)"

# Exit with appropriate code for blender.sh to track success/failure
exit $EXIT_CODE
