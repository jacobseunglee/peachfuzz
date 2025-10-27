#!/bin/bash

# Reset checker script - tests connectivity to scanned ports using netcat
# Reads port mappings from scan.sh output and tests each port
# Stops on first successful connection per host, reports "possible box reset" if all fail
# Usage: ./portchecker.sh [-v|--verbose] <host>

# Default values
VERBOSE=false
HOST=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -*)
            echo "Unknown option: $1"
            echo "Usage: $0 [-v|--verbose] <host>"
            exit 1
            ;;
        *)
            if [[ -z "$HOST" ]]; then
                HOST="$1"
            else
                echo "Error: Multiple host arguments provided"
                echo "Usage: $0 [-v|--verbose] <host>"
                exit 1
            fi
            shift
            ;;
    esac
done

# Strip any newlines and whitespace from host
HOST=$(echo "$HOST" | tr -d '\n\r' | xargs)
RESULTS_FILE="scan_results/scan_port_mappings.txt"
TIMEOUT=5

if [[ -z "$HOST" ]]; then
    echo "Error: Host argument is required"
    echo "Usage: $0 <host>"
    exit 1
fi

# Check if results file exists
if [[ ! -f "$RESULTS_FILE" ]]; then
    echo "Error: Results file '$RESULTS_FILE' not found"
    echo "Run scan.sh first to generate port mappings"
    exit 1
fi

# Function to log messages with timestamp
log() {
    echo "[$(date '+%H:%M:%S')] $*"
}

# Function for verbose logging
verbose_log() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo "[$(date '+%H:%M:%S')] [VERBOSE] $*"
    fi
}

# Function to report success (verbose only)
verbose_success() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo "✓ $*"
    fi
}

# Function to report failure (verbose only)
verbose_failure() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo "✗ $*"
    fi
}

verbose_log "Checking connectivity for host: $HOST"

# Debug: Show what we're looking for and what's in the file (verbose only)
verbose_log "Looking for host entry: '^$HOST:'"
verbose_log "Available entries in $RESULTS_FILE:"
if [[ -f "$RESULTS_FILE" ]] && [[ "$VERBOSE" == "true" ]]; then
    head -5 "$RESULTS_FILE" | while read -r line; do
        verbose_log "  $line"
    done
elif [[ "$VERBOSE" == "true" ]]; then
    verbose_log "  File does not exist"
fi

# Find the host entry in the results file using more flexible matching
HOST_ENTRY=$(grep -F "$HOST:" "$RESULTS_FILE" 2>/dev/null)

if [[ -z "$HOST_ENTRY" ]]; then
    verbose_failure "Host $HOST not found in scan results"
    if [[ "$VERBOSE" == "true" ]]; then
        verbose_log "Debugging information:"
        verbose_log "Host searched: '$HOST'"
        verbose_log "File exists: $([ -f "$RESULTS_FILE" ] && echo "yes" || echo "no")"
        if [[ -f "$RESULTS_FILE" ]]; then
            verbose_log "File contents:"
            cat "$RESULTS_FILE" | while read -r line; do
                verbose_log "  '$line'"
            done
        fi
    fi
    exit 1
fi

# Extract ports from the entry (format: HOST:PORT1,PORT2,PORT3 or HOST: for no ports)
PORTS=$(echo "$HOST_ENTRY" | cut -d':' -f2)

if [[ -z "$PORTS" ]]; then
    echo "$HOST (no ports found)"
    exit 1
fi

verbose_log "Testing ports: $PORTS"

# Convert comma-separated ports to array
IFS=',' read -ra PORT_ARRAY <<< "$PORTS"

# Test each port with netcat
CONNECTED=false

for port in "${PORT_ARRAY[@]}"; do
    if [[ -n "$port" ]]; then
        
        # Use netcat to test connection
        # -z: Zero-I/O mode (just test connection)
        # -v: Verbose output
        # -w: Timeout in seconds
        if timeout "$TIMEOUT" nc -z -w "$TIMEOUT" "$HOST" "$port" 2>/dev/null; then
            verbose_success "Connection established to $HOST:$port"
            CONNECTED=true
            break
        else
            verbose_failure "Failed to connect to $HOST:$port"
        fi
    fi
done

if [[ "$CONNECTED" == "false" ]]; then
    echo "[POSSIBLE BOX RESET] $HOST (ALL PORTS FAILED!!!)"
    exit 1
else
    exit 0
fi
