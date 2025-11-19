#!/usr/bin/env python3
"""
Simple TeamBlender - Basic team iteration
"""

import yaml
from pathlib import Path
import scan
import json
from portchecker import check_ports
from concurrent.futures import ThreadPoolExecutor
import time
import threading
from datetime import datetime

class Blender:
    
    def __init__(self, config_file="blender_config.yaml"):
        self.config_file = config_file
        self.config = self.load_config()
        self.host_pattern = self.config['hosts']['host_pattern']
        self.teams = list(range(self.config['hosts']['teams']['start'], self.config['hosts']['teams']['end'] + 1))
        self.discovery_ports = self.config['scan']['discovery_ports']
        self.ports = self.config['scan']['ports']
        self.ports_file = self.config['scan']['file']
        self.hosts_file = self.config['hosts']['file']
        self.reference_team = self.config['hosts']['reference_team']
        self.timeout = self.config['port_checker'].get('timeout', 3)
        self.placeholder = self.config['hosts'].get('placeholder', 'team')
        self.max_workers = self.config['execution'].get('max_workers', 4)
        self.reference_subnet = self.host_pattern.format(**{self.placeholder: self.reference_team})
        self.hosts = []
        self.ports = {}
        self.stop_event = threading.Event()
        self.down_hosts = set()  # Centralized set of (team, host) tuples for down hosts
        self.down_hosts_lock = threading.Lock()  # Thread-safe access to down_hosts
        self.report_interval = self.config.get('report_interval', 30)  # Report every 30 seconds
    
    def load_config(self):
        """Load configuration from YAML file"""
        config_path = Path(self.config_file)
        with open(config_path, 'r') as f:
            return yaml.safe_load(f)


    def generate_hosts_file(self, hosts):
        try:
            unformatted_hosts = [self.unformat(host, self.reference_team) for host in hosts]
            # Generate all hosts for all teams
            
            # Backup existing file before overwriting
            if Path(self.hosts_file).exists():
                backup_file = f"{self.hosts_file}.backup"
                # Remove existing backup file if it exists
                if Path(backup_file).exists():
                    Path(backup_file).unlink()
                Path(self.hosts_file).rename(backup_file)
                print(f"Backed up existing {self.hosts_file} to {backup_file}")
           
            with open(self.hosts_file, 'w') as f:
                for host in unformatted_hosts:
                    f.write(f"{host}\n")
            return hosts
            
        except Exception as e:
            print(f"Error generating hosts: {e}")
            return []
    
    def load_hosts_file(self):
        try:
            with open(self.hosts_file, 'r') as f:
                hosts = [line.strip() for line in f if line.strip() and not line.startswith('#')]
            self.hosts = hosts
        except Exception as e:
            print(f"Error loading hosts file: {e}")
            self.hosts = []
    
    def generate_ports_file(self, ports):
        try:
            unformatted_ports = {self.unformat(host, self.reference_team): port_info for host, port_info in ports.items()}
            if Path(self.ports_file).exists():
                backup_file = f"{self.ports_file}.backup"
                # Remove existing backup file if it exists
                if Path(backup_file).exists():
                    Path(backup_file).unlink()
                Path(self.ports_file).rename(backup_file)
                print(f"Backed up existing {self.ports_file} to {backup_file}")
            with open(self.ports_file, 'w') as f:
                json.dump(unformatted_ports, f, indent=2)
            return True
        except Exception as e:
            print(f"Error generating ports file: {e}")
            return False
    def load_ports_file(self):
        try:
            with open(self.ports_file, 'r') as f:
                ports = json.load(f)
            self.ports = ports
        except Exception as e:
            print(f"Error loading ports file: {e}")
            self.ports = {}    

    def unformat(self, formatted_string, format_string):
        unformatted_string = ''
        format_string = str(format_string)
        i = 0
        try:
            while i < len(formatted_string):
                x = formatted_string[i]
                y = self.host_pattern[i]
                if x == y:
                    unformatted_string += x
                    i += 1
                else:
                    x = formatted_string[i+len(format_string)]
                    j = i
                    z = self.host_pattern[j]
                    print(x)
                    while z != x:
                        unformatted_string += z
                        j += 1
                        z = self.host_pattern[j]
                    unformatted_string += formatted_string[i+len(format_string):]
                    break
        except Exception as e:
            print(f"Error unformatting string: {e}")
        return unformatted_string

    def check_host_continuous(self, host_ports_pair):
        """Continuously check a single host across all teams until stop event is set"""
        host, ports = host_ports_pair
        
        while not self.stop_event.is_set():
            try:
                for team in self.teams:
                    if self.stop_event.is_set():
                        break
                    
                    formatted_host = host.format(**{self.placeholder: team})
                    open_port = check_ports(formatted_host, ports)
                    host_team_tuple = (team, formatted_host)
                    
                    with self.down_hosts_lock:
                        if not open_port:
                            # Host is down
                            if host_team_tuple not in self.down_hosts:
                                # New down host - add to set and print alert
                                self.down_hosts.add(host_team_tuple)
                                print(f"TEAM {team} - HOST {formatted_host} - POSSIBLE BOX RESET")
                            # If already in set, don't print duplicate alert
                        else:
                            # Host is up
                            if host_team_tuple in self.down_hosts:
                                # Host recovered - remove from set and print recovery
                                self.down_hosts.remove(host_team_tuple)
                                print(f"TEAM {team} - HOST {formatted_host} - RECOVERED")
                
                # Small delay between checks to prevent overwhelming the network
                if not self.stop_event.wait(0.1):  # Non-blocking wait with 100ms timeout
                    continue
                else:
                    break
                    
            except Exception as e:
                print(f"Error checking host {host}: {e}")
                # Continue checking even if there's an error
                if not self.stop_event.wait(1):  # Wait 1 second before retrying
                    continue
                else:
                    break

    def report_down_hosts(self):
        """Periodically report the current status of down hosts"""
        last_report_time = time.time()
        
        while not self.stop_event.is_set():
            time.sleep(1)  # Check every second
            
            current_time = time.time()
            if current_time - last_report_time >= self.report_interval:
                timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                with self.down_hosts_lock:
                    if self.down_hosts:
                        print(f"\n=== STATUS REPORT [{timestamp}]: {len(self.down_hosts)} hosts currently down ===")
                        for team, host in sorted(self.down_hosts):
                            print(f"  TEAM {team} - {host}")
                        print("=" * 50)
                    else:
                        print(f"\n=== STATUS REPORT [{timestamp}]: All hosts are UP ===")
                
                last_report_time = current_time


        
    def discover_and_scan(self):
        try:
            print(f"Using reference subnet {self.reference_subnet} for discovery...")
            hosts = scan.scan_subnet(self.reference_subnet, self.discovery_ports)
            self.generate_hosts_file(hosts)
            ports = scan.scan_hosts(hosts, self.ports)
            self.generate_ports_file(ports)
        except Exception as e:
            print(f"Error during discover and scan: {e}")
            return False
        return True


    def reset_check(self):
        try:
            self.load_ports_file()
            print(f"Loaded ports for {len(self.ports)} hosts from {self.ports_file}")
        except Exception as e:
            print(f"Error during reset check: {e}")
            return
        
        # Clear any previous events
        self.stop_event.clear()
        
        # Clear the down hosts set for a fresh start
        with self.down_hosts_lock:
            self.down_hosts.clear()
        
        print(f"Starting reset check monitoring with {self.max_workers} threads... (Press Ctrl+C to stop)")
        print(f"Status reports will be generated every {self.report_interval} seconds")
        
        # Start threads for each host
        executor = ThreadPoolExecutor(max_workers=self.max_workers + 1)  # +1 for reporter thread
        host_ports_pairs = list(self.ports.items())
        
        try:
            # Submit all host monitoring tasks
            futures = [executor.submit(self.check_host_continuous, pair) for pair in host_ports_pairs]
            
            # Submit the periodic reporting task
            reporter_future = executor.submit(self.report_down_hosts)
            futures.append(reporter_future)
            
            print(f"Started {len(futures)-1} monitoring threads + 1 reporter thread")
            print("Monitoring active hosts... Press Ctrl+C to stop")
            
            # Poll for completion or interruption
            while True:
                # Check if any thread has finished (shouldn't happen unless error)
                finished_futures = [f for f in futures if f.done()]
                
                if finished_futures:
                    print(f"Warning: {len(finished_futures)} threads finished unexpectedly")
                    for future in finished_futures:
                        try:
                            future.result()  # Get any exceptions
                        except Exception as e:
                            print(f"Thread error: {e}")
                
                # Sleep briefly before next poll
                with self.down_hosts_lock:
                    if self.down_hosts:
                        print("---------- HOSTS DOWN STATUS ----------")
                        for team, host in sorted(self.down_hosts):
                            print(f"  TEAM {team} - {host}")
                time.sleep(30)
                
        except KeyboardInterrupt:
            print("\nReset check stopped by user (Ctrl+C)")
            self._shutdown_threads(futures, "User interruption")
            
        finally:
            # Ensure executor is properly shutdown
            executor.shutdown(wait=True)

    def _shutdown_threads(self, futures, reason):
        """Helper method to gracefully shutdown all threads"""
        print(f"Stopping all monitoring threads due to: {reason}")
        
        # Signal all threads to stop
        self.stop_event.set()
        
        # Final status report before shutdown
        with self.down_hosts_lock:
            if self.down_hosts:
                print(f"\nFINAL STATUS: {len(self.down_hosts)} hosts were down at shutdown:")
                for team, host in sorted(self.down_hosts):
                    print(f"  TEAM {team} - {host}")
        
        # Wait for all threads to finish
        print("Waiting for threads to complete...")
        for i, future in enumerate(futures):
            try:
                future.result(timeout=5)  # Wait up to 5 seconds per thread
            except Exception as e:
                print(f"Thread {i+1} shutdown error: {e}")
        
        print("All threads stopped. Exiting gracefully...")
    
def main():
    import argparse
    
    parser = argparse.ArgumentParser(description='Blender - Host Discovery and Reset Checker')
    parser.add_argument('--config', '-c', default='blender_config.yaml', help='Configuration file path')
    parser.add_argument('--discover-and-scan', '-d', action='store_true', help='Run discovery and scan')
    parser.add_argument('--reset-check', '-r', action='store_true', help='Run reset check on existing hosts')
    parser.add_argument('--verbose', '-v', action='store_true', help='Enable verbose output')
    
    args = parser.parse_args()
    
    # Check if config file exists
    if not Path(args.config).exists():
        print(f"Error: Config file '{args.config}' not found")
        return
    
    blender = Blender(args.config)
    
    print("Blender")
    print("=" * 40)

    if args.discover_and_scan:
        print("Running discovery and scan...")
        success = blender.discover_and_scan()
        if success:
            print("\nDiscovery and scan completed successfully!")
        else:
            print("\nDiscovery and scan failed!")
    if args.reset_check:
        print("Running reset check...")
        blender.reset_check()
    else:
        # Default behavior - show help
        parser.print_help()
        print("\nExamples:")
        print("  python blender.py --discover-and-scan")
        print("  python blender.py --reset-check")
        print("  python blender.py --config custom.yaml --discover-and-scan")

if __name__ == "__main__":
    main()