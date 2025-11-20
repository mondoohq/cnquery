## ipinfo provider for cnquery

This provider makes public IP address information accessible from ipinfo.io.

It provides a connector for querying IP information via the ipinfo.io API.

Examples:
  cnquery shell ipinfo
  cnquery run -c "ipinfo(ip('8.8.8.8')){*}"
  cnquery run -c "ipinfo(){*}"  # Query your public IP"
  cnquery run -c "network.interfaces.map(ips.map(_.ip)).flat.map(ipinfo(_){*})"

