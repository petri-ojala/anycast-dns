package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"time"

	api "github.com/osrg/gobgp/api"
	gobgp "github.com/osrg/gobgp/pkg/server"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var config *anycastConfig
var anycast *anycastType

// Configuration file
var ConfigFile = flag.String("config", "/usr/local/etc/anycast-dns.json", "Configuration file")

func main() {
	// Anycast routing state
	anycast = &anycastType{}
	anycast.state = false
	anycast.bgpState = ""

	log.SetFormatter(&log.JSONFormatter{})

	flag.Parse()

	// Read configuration file
	config = &anycastConfig{}
	c, err := ioutil.ReadFile(*ConfigFile)
	if err == nil {
		json.Unmarshal(c, &config)
	} else {
		log.Fatal(err)
	}
	// Parse durations
	config.DNS.Interval, _ = time.ParseDuration(config.DNS.IntervalString)
	config.DNS.Timeout, _ = time.ParseDuration(config.DNS.TimeoutString)

	// Get nexthop IP's from the interface
	config.getNextHopIP()

	// Create BGP routes from interface
	anycast.getInterfaces(config.Anycast.Interface, config.Anycast.Prefix)

	// Create BGP routes from static routes
	anycast.addRoutes(config)

	// Initialise BGP
	anycast.s = gobgp.NewBgpServer()
	go anycast.s.Serve()

	// Start BGP
	if err := anycast.s.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: &api.Global{
			As:         config.BGP.MyAS,
			RouterId:   config.BGP.RouterID,
			ListenPort: 179, // gobgp won't listen on tcp:179
		},
	}); err != nil {
		log.Fatal(err)
	}

	// Monitor the change of the BGP peer state
	if err := anycast.s.MonitorPeer(context.Background(), &api.MonitorPeerRequest{}, func(p *api.Peer) { anycast.updateRoute(); anycast.bgpState = p.String(); log.Info(p) }); err != nil {
		log.Error(err)
		return
	}

	// Configure BGP neighbors
	//
	// IPv4 and IPv6 unicast address families are configured
	for _, p := range config.BGP.Peers {
		n := &api.Peer{
			Conf: &api.PeerConf{
				NeighborAddress: p.IPAddress,
				PeerAs:          p.AS,
			},
			AfiSafis: []*api.AfiSafi{
				{
					Config: &api.AfiSafiConfig{
						Family: &api.Family{
							Afi:  api.Family_AFI_IP,
							Safi: api.Family_SAFI_UNICAST,
						},
						Enabled: true,
					},
				},
				{
					Config: &api.AfiSafiConfig{
						Family: &api.Family{
							Afi:  api.Family_AFI_IP6,
							Safi: api.Family_SAFI_UNICAST,
						},
						Enabled: true,
					},
				},
			},
		}

		if err := anycast.s.AddPeer(context.Background(), &api.AddPeerRequest{
			Peer: n,
		}); err != nil {
			log.Fatal(err)
		}
	}

	log.Info("DNS Anycast routing started")

	for true {
		// Loop forever

		dnsState := checkDNS()

		if dnsState != anycast.state {
			// State changed
			anycast.state = dnsState
			anycast.updateRoute()
		}

		time.Sleep(config.DNS.Interval)
	}
}
