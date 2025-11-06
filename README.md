# PeachFuzz - Host Discovery and Team Iteration Framework

A comprehensive Python framework for automated host discovery, team iteration, and network scanning. Originally developed as bash scripts, now evolved into a production-ready Python system with decorator patterns and YAML configuration.

## 🚀 Key Features

### Host Discovery System
- **Automatic host discovery** from scanning one reference team
- **Network range scanning** with configurable IP ranges
- **Hostname pattern testing** with DNS resolution
- **Port connectivity verification** during discovery
- **Automatic hosts.txt generation** with template expansion

### Team Iteration Framework
- **Decorator-based architecture** for clean team processing
- **YAML configuration management** for flexible settings
- **Multiple team range support** with inclusive ranges
- **Host template expansion** with `_` and `{team}` placeholders
- **Parallel processing capabilities** (future enhancement)

### Integrated Scanning & Port Checking
- **RustScan integration** with ultra-fast port scanning
- **Cross-platform binary detection** (Windows/Linux/macOS)
- **Socket-based port connectivity testing**
- **Result aggregation and reporting**
- **High-performance batch scanning**

## 📁 Project Structure

```
peachfuzz/
├── blender.py                    # Core decorator class with discovery
├── blender_config.yaml           # Discovery-enabled configuration
├── discovery.py                  # Standalone discovery script
├── discovery_example.py          # Discovery examples and demos
├── scan.py                       # Enhanced scanner with discovery
├── portchecker.py                # Port connectivity testing
├── hosts.txt                     # Generated hosts file (auto-created)
├── requirements.txt              # Python dependencies
└── README.md                     # This file
```

## 🔧 Installation & Setup

### Prerequisites
- Python 3.6+ with type hints support
- RustScan (included binaries for Windows/Linux/macOS)
- PyYAML (automatically installed)

### Installation
```bash
# Clone or download the project
cd peachfuzz

# Install Python dependencies
pip install -r requirements.txt

# Verify RustScan binaries (included in ./scanning/ directory)
ls ./scanning/
```

## 🎯 Quick Start

### 1. Host Discovery Workflow

Generate a hosts.txt file by discovering hosts from one reference team:

```bash
# Run discovery with default settings
python discovery.py --verbose

# Use custom reference team
python discovery.py --team 5 --verbose

# Test configuration without running discovery
python discovery.py --test-config
```

### 2. Using the Decorator System

```python
from blender import blender

@blender("blender_config.yaml")
def process_teams(team: int, hosts: list):
    """Process each team with its expanded hosts"""
    print(f"Processing team {team}")
    for host in hosts:
        print(f"  Working on {host}")
    return {"team": team, "processed": len(hosts)}

# Execute across all teams
results = process_teams()
```

### 3. Integrated Scanning

```bash
# Discover hosts and scan all teams
python scan.py --discover-and-scan --verbose

# Scan specific ports across all teams
python scan.py --scan-teams --ports 22,80,443

# Scan single host
python scan.py example.com --verbose
```

## ⚙️ Configuration

### Discovery Configuration

The `blender_config.yaml` file controls all discovery and processing settings:

```yaml
# Host discovery settings
discovery:
  enabled: true                     # Enable auto-discovery
  reference_team: 1                 # Team to scan for patterns
  network_ranges:                   # IP ranges to scan
    - "192.168.1.0/24"
    - "10.0.0.0/24"
  host_patterns:                    # Hostname patterns to test
    - "team{team}.example.com"
    - "ctf{team}.local"
    - "db{team}.internal"
  common_ports: [22, 80, 443, 3389] # Ports to check
  output_file: "hosts.txt"          # Generated hosts file

# Team configuration
teams:
  start: 1                          # Starting team number
  end: 10                           # Ending team number
  ranges:                           # Multiple ranges support
    - start: 1
      end: 5
    - start: 10
      end: 15

# Host configuration
hosts:
  file: "hosts.txt"                 # Hosts file to read
  auto_generate: true               # Auto-run discovery if missing
  templates:                        # Fallback templates
    - "team_.example.com"
    - "192.168._.100"
```

## 🔍 Host Discovery Methods

### 1. Network Range Scanning
Scans IP ranges for live hosts with open ports:
```python
network_ranges = ["192.168.1.0/24", "10.0.0.0/24"]
```

### 2. Hostname Pattern Testing
Tests hostname patterns with team substitution:
```python
host_patterns = [
    "team{team}.example.com",     # team1.example.com, team2.example.com
    "ctf{team}.local",            # ctf1.local, ctf2.local
    "db{team}.internal"           # db1.internal, db2.internal
]
```

### 3. Template Generation
Automatically converts discovered hosts to templates:
- `team1.example.com` → `team_.example.com`
- `192.168.1.100` → `192.168._.100`

## 🏃‍♂️ Usage Examples

### Discovery Examples

```bash
# Basic discovery with verbose output
python discovery.py --verbose

# Custom configuration and reference team
python discovery.py --config my_config.yaml --team 3

# Test configuration without discovery
python discovery.py --test-config --config blender_config.yaml
```

### Decorator Examples

```python
# Simple team processing
@blender()
def ping_teams(team: int, hosts: list):
    for host in hosts:
        print(f"Pinging {host} for team {team}")

# With custom configuration
@blender("custom_config.yaml")
def advanced_processing(team: int, hosts: list, extra_param="default"):
    return {"team": team, "hosts": len(hosts), "param": extra_param}

# Manual iteration (non-decorator)
blender_instance = TeamBlender("config.yaml")
team_hosts = blender_instance._get_all_hosts_for_teams()
for team, hosts in team_hosts.items():
    print(f"Team {team}: {hosts}")
```

### CLI Usage

```bash
# Built-in commands
python blender.py --discover --verbose          # Run discovery
python blender.py --list-teams                  # Show all teams/hosts
python blender.py --test-team 1 --verbose       # Test team connectivity

# Enhanced scanning
python scan.py --scan-teams --verbose  # Scan all teams
python scan.py host.example.com        # Scan single host
```

## 🔗 Integration with Existing Tools

### Port Checker Integration
```python
from portchecker import PortChecker

@blender()
def check_team_ports(team: int, hosts: list):
    checker = PortChecker(timeout=5)
    for host in hosts:
        open_ports = checker.check_host_ports(host, [22, 80, 443])
        print(f"Team {team} - {host}: {open_ports}")
```

### RustScan Integration
The enhanced scanner uses RustScan for ultra-fast port discovery with cross-platform binary detection. RustScan provides significant performance improvements over traditional scanners and can be configured through the YAML settings.

## 🎨 Advanced Features

### Auto-Discovery Workflow
1. **Configuration Check**: Verifies discovery settings
2. **Network Scanning**: Scans configured IP ranges
3. **Pattern Testing**: Tests hostname patterns with DNS resolution
4. **Template Generation**: Converts found hosts to reusable templates
5. **File Creation**: Generates organized hosts.txt with comments

### Flexible Team Ranges
```yaml
teams:
  ranges:
    - start: 1      # Teams 1-5
      end: 5
    - start: 10     # Teams 10-15  
      end: 15
    - start: 100    # Teams 100-199
      end: 199
```

### Error Handling & Validation
- **Configuration validation** with helpful error messages
- **Graceful failure handling** with retry mechanisms
- **Comprehensive logging** with verbosity controls
- **Cross-platform compatibility** testing

## 🐛 Troubleshooting

### Common Issues

**Discovery finds no hosts:**
- Check network ranges are accessible
- Verify hostname patterns match your environment
- Ensure reference team exists and is reachable
- Check firewall and network connectivity

**RustScan binary not found:**
```bash
# Ensure RustScan binaries are present in ./scanning/ directory
# The project includes pre-compiled binaries for:
# - Windows: rustscan-windows.exe
# - Linux: rustscan-linux  
# - macOS: rustscan-macos
# 
# If missing, download from: https://github.com/RustScan/RustScan/releases
```

**Configuration errors:**
```bash
# Test configuration
python discovery.py --test-config

# Check YAML syntax
python -c "import yaml; yaml.safe_load(open('blender_config.yaml'))"
```

### Debug Mode

Enable verbose output for detailed debugging:
```bash
python discovery.py --verbose
python blender.py --discover --verbose
python scan.py --scan-teams --verbose
```

## 🔄 Migration from Bash Scripts

The original bash scripts (`blender.sh`, `scan.sh`, `portchecker.sh`) have been completely rewritten in Python with enhanced functionality:

- **bash blender.sh --hosts hosts.txt --teams 1 5** → `@blender() decorator`
- **bash scan.sh host** → `python scan.py host`
- **bash portchecker.sh** → `PortChecker class integration`

All original functionality is preserved while adding:
- Type safety and error handling
- YAML configuration management
- Object-oriented architecture
- Cross-platform compatibility
- Automated host discovery

## 📚 API Reference

### TeamBlender Class
- `discover_and_generate_hosts_file()` - Run host discovery
- `_get_all_hosts_for_teams()` - Get expanded hosts for all teams
- `_ping_host(host)` - Test host connectivity
- `_scan_network_range(range, ports)` - Scan IP range

### Decorator Usage
```python
@blender(config_path)
def function_name(team: int, hosts: List[str], **kwargs):
    # Process team and hosts
    return results
```

### Configuration Structure
See `blender_config.yaml` for complete configuration options with inline documentation.

## 🤝 Contributing

This framework evolved from simple bash scripts to a comprehensive Python system. Contributions welcome for:
- Additional discovery methods
- Enhanced scanning capabilities  
- Performance optimizations
- Cross-platform testing
- Documentation improvements

## 📄 License

Open source project developed for security testing and network reconnaissance. Use responsibly and in accordance with applicable laws and policies.