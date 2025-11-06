#!/usr/bin/env python3
"""
Simple RustScan wrapper
"""

import subprocess
import sys
import platform
import pathlib
from pathlib import Path

def get_rustscan_binary():
    """Get the appropriate RustScan binary for the current OS"""
    suffixes = {
        'Windows': 'windows.exe',
        'Linux': 'linux',
        'Darwin': 'macos',
    }
    sys_name = platform.system()
    suffix = suffixes.get(sys_name)
    if suffix is None:
        raise NotImplementedError(f'Platform not supported: {sys_name}')
    
    rustscan_path = pathlib.Path(f'./scanning/rustscan-{suffix}')
    if rustscan_path.exists():
        return rustscan_path
    
    raise FileNotFoundError(f'RustScan binary not found for {sys_name}')

def scan_subnet(subnet, ports):
    
    # Get binary
    rustscan_path = get_rustscan_binary()
    
    # Build command
    cmd = [str(rustscan_path), '-a', subnet, '-g', '-p', ",".join(map(str, ports))]
    
    print(f"Running: {' '.join(cmd)}")
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=300)
        
        if result.returncode == 0:
            # Parse results
            hosts = []
            for line in result.stdout.splitlines():
                line = line.strip()
                if '->' in line:
                    parts = line.split('->')
                    host = parts[0].strip()
                    hosts.append(host)
            return hosts
        else:
            print(f"Error: {result.stderr}")
            return []
    except subprocess.TimeoutExpired:
        print("Scan timed out")
        return []
    except Exception as e:
        print(f"Error running scan: {e}")
        return []


def scan_hosts(hosts, ports):
    rustscan_path = get_rustscan_binary()
    results = {}
    for host in hosts:
        cmd = [str(rustscan_path), '-a', host, '-g', "".join(map(str, ports)) if ports else '--top']
        print(f"Running: {' '.join(cmd)}")
        try:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=300)

            if result.returncode == 0:
                # Parse results
                for line in result.stdout.splitlines():
                    line = line.strip()
                    if '->' in line:
                        parts = line.split('->')
                        if len(parts) == 2:
                            host = parts[0].strip()
                            port_section = parts[1].strip()

                            if port_section.startswith('[') and port_section.endswith(']'):
                                port_str = port_section[1:-1]
                                if port_str:
                                    try:
                                        ports_found = [int(p.strip()) for p in port_str.split(',')]
                                        results[host] = sorted(ports_found)
                                    except ValueError:
                                        pass
        except subprocess.TimeoutExpired:
            print("Scan timed out") 
        except Exception as e:
            print(f"Error running scan: {e}")
    return results

def main():
    """Main function"""
    if len(sys.argv) < 2:
        print("Usage: python scan.py <subnet>")
        print("Example: python scan.py 10.188.1.0/24")
        sys.exit(1)
    
    subnet = sys.argv[1]
    print(f"Scanning {subnet}...")
    
    hosts = scan_subnet(subnet)
    results = scan_hosts([host for host in hosts])
    if results:
        print(f"\nFound {len(results)} hosts:")
        for host, ports in results.items():
            print(f"{host}: {ports}")
    else:
        print("No hosts found")

if __name__ == "__main__":
    main()