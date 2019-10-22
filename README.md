# anycast-dns

A simple daemon to monitor DNS service and announce BGP routes to implement anycast DNS service

Anycast routes are supported for both IPv4 and IPv6 with a calculated or static LocalPref.

BGP peering has been tested with Ubiquiti EdgeRouter and `gobgpd`.  For the peers only IPv4 Unicast and IPv6 Unicast protocols are defined.

## Configuration file

```json
{
	"dns": {
		"ip": "127.0.0.1",
		"port": "53",
		"query": "google.com.",
		"timeout": "1s",
		"interval": "1s"
	},
	"anycast": {
		"interface": "lo",
		"prefix": "dns",
		"localpref": 200,
		"decrease": 10,
		"nexthop-interface": "eth0",
		"routes": [
			{
				"label": "v6-primary",
				"ip": "fd00::8:8:8:4/128",
				"localpref": 200
			},
			{
				"label": "v6-secondary",
				"ip": "fd00::8:8:4:4/128",
				"localpref": 190
			}
		]
	},
	"bgp": {
		"id": "172.17.2.1",
		"as": 64512,
		"peers": [
			{
				"ip": "172.17.2.254",
				"description": "Edgerouter",
				"as": 64512
			},
			{
				"ip": "172.17.2.253",
				"description": "gobgpd",
				"as": 64512
			}
		]
	}
}
```

`dns` defines the health check for the DNS service.  `ip` and `port´ define the DNS server IP and port for the DNS query.  `query` defines the domain to query, the RR type is hardcoded `SOA`.  `timeout` defines the timeout for the query (as `time.Duration`), `interval` defines the health check interval.

`anycast` defines BGP route announcements.  There are two sources available, interfaces with labels and static announcements.  Labeled interfaces support only IPv4 addresses.  For example `interface` with `lo` and `prefix` with `dns` will look for any `lo:dns*` interfaces.  First interface will get BGP LocalPref of `localpref`, and it is decreased by `decrease` for each subsequent interface.  
`routes` defines any static routes to announce, both IPv4 and IPv6 are supported.  `label` is a description for the route, `ip` is the IP prefix to announce (only `/32` and `/128` are supported), and `localpref` defines the BGP LocalPref for the route.  
`nexthop-interface` defines the NextHop for the BGP route announcements.

`bgp` defines the BGP server and peers.  `id` defines BGP Server ID and `as` defines our AS number.  `peers` lists all BGP peers, `ip` is the peer's IP address, `description` is a description for the BGP peering, and `as` is the remote AS number.

## Use case

I'm running this configuration in my home network to provide resilient and redundant DNS service.  I have I configured two DNS servers that server DNS with 10.10.10.10 and fd00::10:10:10:10 IP address.  Both are running the `anycast-dns` daemon with different metric configuration.  In case the primary server fails, the secondary server will take over immediately.

I have also a third DNS service running in a Kubernetes cluster that is using `metallb` L3 BGP routing to announce itself as 10.10.10.10 with lower LocalPref.  In case the baremetal DNS servers fail, there might still be DNS service available from the Kubernetes cluster.

DHCP is giving 10.10.10.10 as the DNS server, SLAAC is giving the fd00::10:10:10:10 as IPv6 DNS address (RFC 8106).  I do not use DHCPv6.

## Packages

This tool is built on top of these two packages:

* `github.com/osrg/gobgp` for the BGP implementation [https://osrg.github.io/gobgp/](https://osrg.github.io/gobgp/)

* `https://github.com/insomniacslk/dhcp` for the DHCPv4 implementation, used e.g. by Facebook in `dhcplb`

