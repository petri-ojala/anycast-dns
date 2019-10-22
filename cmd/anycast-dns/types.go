package main

import (
	"net"
	"time"

	"github.com/golang/protobuf/ptypes/any"
	gobgp "github.com/osrg/gobgp/pkg/server"
	"github.com/vishvananda/netlink"
)

const (
	RT_SCOPE_UNIVERSE = 0x0
)

// Configuration file content
type anycastConfig struct {
	DNS struct {
		ServerIP       string `json:"ip"`
		ServerPort     string `json:"port"`
		Query          string `json:"query"`
		Timeout        time.Duration
		TimeoutString  string `json:"timeout"`
		Interval       time.Duration
		IntervalString string `json:"interval"`
	} `json:"dns"`
	Anycast struct {
		Interface        string `json:"interface"`
		Prefix           string `json:"prefix"`
		LocalPref        uint32 `json:"localpref"`
		LocalPrefDec     uint32 `json:"decrease"`
		NextHopInterface string `json:"nexthop-interface"`
		NextHopV4        net.IP
		NextHopV6        net.IP
		Routes           []ipRoute `json:"routes"`
	} `json:"anycast"`
	BGP struct {
		RouterID string    `json:"id"`
		MyAS     uint32    `json:"as"`
		Peers    []BGPPeer `json:"peers"`
	} `json:"bgp"`
}

type BGPPeer struct {
	IPAddress   string `json:"ip"`
	Description string `json:"description"`
	AS          uint32 `json:"as"`
}

type ipRoute struct {
	Label     string `json:"label"`
	IP        string `json:"ip"`
	LocalPref uint32 `json:"localpref"`
}

// Internal struct for current state of the world
type anycastType struct {
	state      bool
	bgpState   string
	routeTable []*anycastRoute
	s          *gobgp.BgpServer
}

type anycastRoute struct {
	ifName        string
	ifLabel       string
	ipAddr        *net.IPNet
	ifDetail      netlink.Addr
	LocalPref     uint32
	bgpNLRI       *any.Any
	bgpAttributes []*any.Any
}
