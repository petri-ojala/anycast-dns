package main

import (
	"strings"

	"github.com/vishvananda/netlink"
	log "github.com/sirupsen/logrus"
)

//
// Get net.IP's for nexthop interface
func (c *anycastConfig) getNextHopIP() error {
	// Get Inteface by name
	ifList, err := netlink.LinkByName(c.Anycast.NextHopInterface)
	if err != nil {
		return err
	}

	// Get all addresses for Interface
	ifAddrs, err := netlink.AddrList(ifList, netlink.FAMILY_ALL)
	if err != nil {
		return err
	}

	for _, a := range ifAddrs {
		if a.Scope == RT_SCOPE_UNIVERSE {
			if a.IPNet.IP.To4() != nil {
				c.Anycast.NextHopV4 = a.IPNet.IP
			} else {
				c.Anycast.NextHopV6 = a.IPNet.IP
			}
		}
	}

	return err
}

//
// Build BGP routes for all interface-based routes (if + label)
func (r *anycastType) getInterfaces(ifName, labelPrefix string) error {
	if ifName == "" {
		return nil
	}

	// Get Inteface by name
	ifList, err := netlink.LinkByName(ifName)
	if err != nil {
		return err
	}

	// Get all addresses for Interface
	ifAddrs, err := netlink.AddrList(ifList, netlink.FAMILY_ALL)
	if err != nil {
		return err
	}

	// ifAddrs result should be sorted by the labels, now the order will be the order they've been created
	// Only IPv4 supported

	l := config.Anycast.LocalPref
	for _, a := range ifAddrs {
		if strings.HasPrefix(a.Label, ifName+":"+labelPrefix) {
			// Matching label, eg. "lo:dns42" will match for "lo" and "dns"
			cidrLength, _ := a.IPNet.Mask.Size()

			nlri, attr := createBGPRoute(a.IPNet, config.Anycast.NextHopV4, l)

			r.routeTable = append(r.routeTable, &anycastRoute{
				ifName:        ifName,
				ifLabel:       a.Label,
				ipAddr:        a.IPNet,
				ifDetail:      a,
				LocalPref:     uint32(l),
				bgpNLRI:       nlri,
				bgpAttributes: attr,
			})

			log.WithFields(log.Fields{
				"interface": a.Label,
				"ip":        a.IPNet.IP.String(),
				"cidr":      cidrLength,
				"localpref": l,
			}).Info("interface added")

			l = l - config.Anycast.LocalPrefDec
		}
	}

	return err
}
