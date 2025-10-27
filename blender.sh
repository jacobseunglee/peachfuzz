#!/bin/bash

# Default values
HOSTS_FILE=""
TEAMS_START=""
TEAMS_END=""
SCRIPT_COMMAND=""

# Function to display usage
usage() {
    echo "Usage: $0 --hosts <hosts_file> --teams <start> <end> '<script with args>'"
    echo "       $0 --hosts <hosts_file> --teams <end> '<script with args>'"
    echo "  --hosts    Path to the hosts file"
    echo "  --teams    Team range - either one number (defaults start to 1) or two numbers (start end)"
    echo "             Replaces _ in host entries with numbers in the specified range (inclusive)"
    echo "  <script>   Script/command to execute for each host (host will be passed as last argument)"
    echo "             Can include arguments, quote if contains spaces"
    echo "Examples:"
    echo "  $0 --hosts hosts.txt --teams 5 ./ping_host.sh"
    echo "  $0 --hosts hosts.txt --teams 3 7 'ssh -o ConnectTimeout=5'"
    echo "  $0 --hosts hosts.txt --teams 5 'ping -c 3'"
    exit 1
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --hosts)
            HOSTS_FILE="$2"
            shift 2
            ;;
        --teams)
            if [[ $# -lt 2 ]]; then
                echo "Error: --teams requires at least one argument"
                usage
            fi
            TEAMS_START="$2"
            # Check if there's a third argument for end range
            if [[ $# -ge 3 ]] && [[ "$3" =~ ^[0-9]+$ ]]; then
                TEAMS_END="$3"
                shift 3
            else
                # Only one number provided, default start to 1 and use provided number as end
                TEAMS_END="$TEAMS_START"
                TEAMS_START="1"
                shift 2
            fi
            ;;
        -h|--help)
            usage
            ;;
        *)
            # If it doesn't start with -, treat it as the script command argument
            if [[ "$1" != -* ]]; then
                if [[ -z "$SCRIPT_COMMAND" ]]; then
                    SCRIPT_COMMAND="$1"
                    shift
                else
                    echo "Error: Multiple script command arguments provided"
                    usage
                fi
            else
                echo "Unknown option: $1"
                usage
            fi
            ;;
    esac
done

# Check if hosts file is provided
if [[ -z "$HOSTS_FILE" ]]; then
    echo "Error: --hosts argument is required"
    usage
fi

# Check if teams arguments are provided
if [[ -z "$TEAMS_START" ]] || [[ -z "$TEAMS_END" ]]; then
    echo "Error: --teams argument is required"
    usage
fi

# Check if script command argument is provided
if [[ -z "$SCRIPT_COMMAND" ]]; then
    echo "Error: script command argument is required"
    usage
fi

# Parse the script command to extract the script name for validation
SCRIPT_NAME=$(echo "$SCRIPT_COMMAND" | awk '{print $1}')

# Check if script exists and is executable
if [[ ! -f "$SCRIPT_NAME" ]]; then
    # Try to find the script in PATH
    SCRIPT_PATH=$(which "$SCRIPT_NAME" 2>/dev/null)
    if [[ -n "$SCRIPT_PATH" && -x "$SCRIPT_PATH" ]]; then
        # Replace the script name in the command with the full path
        SCRIPT_COMMAND="${SCRIPT_COMMAND/$SCRIPT_NAME/$SCRIPT_PATH}"
    else
        echo "Error: Script '$SCRIPT_NAME' not found in current directory or PATH"
        exit 1
    fi
elif [[ ! -x "$SCRIPT_NAME" ]]; then
    chmod +x "$SCRIPT_NAME" 2>/dev/null
    if [[ ! -x "$SCRIPT_NAME" ]]; then
        echo "Error: Could not make script '$SCRIPT_NAME' executable"
        exit 1
    fi
fi

# Validate teams start is a non-negative integer
if ! [[ "$TEAMS_START" =~ ^[0-9]+$ ]]; then
    echo "Error: teams start must be a non-negative integer (got: $TEAMS_START)"
    exit 1
fi

# Validate teams end is a non-negative integer
if ! [[ "$TEAMS_END" =~ ^[0-9]+$ ]]; then
    echo "Error: teams end must be a non-negative integer (got: $TEAMS_END)"
    exit 1
fi

# Validate that start <= end
if [[ $TEAMS_START -gt $TEAMS_END ]]; then
    echo "Error: teams start ($TEAMS_START) must be less than or equal to teams end ($TEAMS_END)"
    exit 1
fi

# Check if hosts file exists
if [[ ! -f "$HOSTS_FILE" ]]; then
    echo "Error: Hosts file '$HOSTS_FILE' not found"
    exit 1
fi

# Check if hosts file is readable
if [[ ! -r "$HOSTS_FILE" ]]; then
    echo "Error: Cannot read hosts file '$HOSTS_FILE'"
    exit 1
fi

# Loop through each host in the file
while IFS= read -r host || [[ -n "$host" ]]; do
    # Skip empty lines and lines starting with #
    if [[ -z "$host" ]] || [[ "$host" =~ ^[[:space:]]*# ]]; then
        continue
    fi
    
    # Remove leading/trailing whitespace
    host=$(echo "$host" | xargs)
    
    # Check if host contains underscore
    if [[ "$host" == *"_"* ]]; then
        # Loop through team numbers from TEAMS_START to TEAMS_END
        for ((team=TEAMS_START; team<=TEAMS_END; team++)); do
            # Replace _ with team number
            expanded_host="${host//_/$team}"
            
            # Execute the command with the expanded host as final argument
            eval "$SCRIPT_COMMAND \"$expanded_host\""
        done
    else
        # Execute the command with the host as final argument
        eval "$SCRIPT_COMMAND \"$host\""
    fi
    
done < "$HOSTS_FILE"
