# nmap provider

Nmap, short for Network Mapper, is a powerful and versatile open-source tool used for network discovery and security auditing. This tool is widely utilized by network administrators, security professionals, and penetration testers to map out network structures, discover hosts, identify services, and detect vulnerabilities.

The nmap provider maps primary objects and attributes that nmap uses to store and manage information about scanned targets, discovered hosts, and their associated ports and services.

## Pre-requisites

This provider requires the nmap tool to be installed on your system. You can download and install nmap from the official [website](https://nmap.org/download.html).

## Get Started

```shell
cnquery shell nmap
```

## Example

*Scan active IP address in network*

```shell
nmap.target("192.168.178.0/24").hosts { name ports { * }  }
nmap.target.hosts: [
  0: {
    ports: [
      0: {
        service: "http"
        version: ""
        method: "probed"
        state: "open"
        protocol: "tcp"
        port: 443
        product: "FRITZ!Box http config"
      }
      1: {
        service: "sip"
        version: ""
        method: "probed"
        state: "open"
        protocol: "tcp"
        port: 5060
        product: "AVM FRITZ!OS SIP"
      }
    ]
    name: "192.168.178.1"
  }
  1: {
    ports: [
      0: {
        service: "rtsp"
        version: "770.8.1"
        method: "probed"
        state: "open"
        protocol: "tcp"
        port: 5000
        product: "AirTunes rtspd"
      }
      1: {
        service: "rtsp"
        version: "770.8.1"
        method: "probed"
        state: "open"
        protocol: "tcp"
        port: 7000
        product: "AirTunes rtspd"
      }
    ]
    name: "192.168.178.25"
  }
]
```

*Host scan with specific ip*

```shell
nmap.target(target: "192.168.178.25").hosts { ports }
nmap.target.hosts: [
  0: {
    ports: [
      0: nmap.port port=5000 service="rtsp"
      1: nmap.port port=7000 service="rtsp"
    ]
  }
]
```

# Advanced Usage

Discover all exposed hosts on a network.

```shell
cnquery shell nmap --networks "192.168.0.0/20" --discover hosts
```

Connect to a specific IP address and display all open ports.

```shell
cnquery shell nmap host 8.8.8.8
```