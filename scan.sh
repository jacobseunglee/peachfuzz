#!/bin/bash

# Nmap output parser for use with blender.sh
# This script runs nmap on a host and parses the output into a structured format
# Usage: ./scan.sh [-v|--verbose] <host>

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

# Strip any newlines from host
HOST=$(echo "$HOST" | tr -d '\n\r')
OUTPUT_DIR="scan_results"
TEMP_RESULTS_FILE="$OUTPUT_DIR/scan_port_mappings.txt"

# Validate that host is provided
if [[ -z "$HOST" ]]; then
    echo "Error: Host argument is required"
    echo "Usage: $0 <host>"
    exit 1
fi

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Function to log messages with timestamp (verbose only)
verbose_log() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo "[$(date '+%H:%M:%S')] $*"
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

verbose_log "Starting nmap scan for host: $HOST"

# Run nmap scan
# -sS: TCP SYN scan (stealthy)
# -T4: Aggressive timing template (faster)
# -p-: Scan all 65535 ports
# --open: Only show open ports
# -oG: Greppable output format for easier parsing
NMAP_OUTPUT="nmap.txt"
NMAP_TEMP="nmap_temp.txt"

verbose_log "Running nmap scan (this may take a while)..."

# Common ports scan first for faster results
if nmap -sS -T4 --top-ports 1000 --open -oG "$NMAP_TEMP" "$HOST" >/dev/null 2>&1; then
    verbose_success "Nmap scan completed for $HOST"
    
    # Parse the nmap output
    verbose_log "Parsing nmap results..."
    
    # Extract open ports from greppable output
    # Format: Host: <ip> (<hostname>)	Ports: <port>/<state>/<protocol>/<service>, ...
    OPEN_PORTS=$(grep "^Host:" "$NMAP_TEMP" | grep "Ports:" | sed 's/.*Ports: //' | tr ',' '\n' | grep "/open/" | cut -d'/' -f1 | sort -n | tr '\n' ',' | sed 's/,$//' | tr -d ' ')
    
    if [[ -n "$OPEN_PORTS" ]]; then
        verbose_success "Found open ports: $OPEN_PORTS"
        
        # Check if host already exists in results file and replace it
        if [[ -f "$TEMP_RESULTS_FILE" ]] && grep -q "^$HOST:" "$TEMP_RESULTS_FILE"; then
            # Host exists, replace the line
            sed -i "s/^$HOST:.*/$HOST:$OPEN_PORTS/" "$TEMP_RESULTS_FILE"
            verbose_log "Updated existing entry for $HOST"
        else
            # Host doesn't exist, append new entry
            echo "$HOST:$OPEN_PORTS" >> "$TEMP_RESULTS_FILE"
            verbose_log "Added new entry for $HOST"
        fi
        EXIT_CODE=0
    fi
fi

# Cleanup temporary files
rm -f "$NMAP_TEMP" "$NMAP_OUTPUT"

verbose_log "Scan completed for $HOST (exit code: $EXIT_CODE)"

exit $EXIT_CODE
