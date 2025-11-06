#!/usr/bin/env python3
"""
Simple Port Checker
Reads hosts.txt and checks port connectivity for all hosts.
"""

import socket

def check_ports(host, ports, timeout=3):
    """Check if a list of ports are open on a host"""
    for port in ports:
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(timeout)
            result = sock.connect_ex((host.strip(), port))
            if result == 0:
                return True
            sock.close()
        except KeyboardInterrupt as e:
            raise e
        except:
            pass
    return False

