package main

import (
	"context"
	"net"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	api "github.com/osrg/gobgp/api"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

//
// Build BGP Route (NLRI and attributes) for announcement
func createBGPRoute(i *net.IPNet, n net.IP, l uint32) (*any.Any, []*any.Any) {
	var a2 *any.Any

	// Build BGP Attributes
	cidrLength, _ := i.Mask.Size()

	nlri, _ := ptypes.MarshalAny(&api.IPAddressPrefix{
		Prefix:    i.IP.String(),
		PrefixLen: uint32(cidrLength),
	})

	a1, _ := ptypes.MarshalAny(&api.OriginAttribute{
		Origin: 0,
	})

	a3, _ := ptypes.MarshalAny(&api.LocalPrefAttribute{
		LocalPref: l,
	})

	if i.IP.To4() != nil {
		a2, _ = ptypes.MarshalAny(&api.NextHopAttribute{
			NextHop: n.String(),
		})
	} else {
		a2, _ = ptypes.MarshalAny(&api.MpReachNLRIAttribute{
			Family: &api.Family{
				Afi:  api.Family_AFI_IP6,
				Safi: api.Family_SAFI_UNICAST,
			},
			NextHops: []string{n.String()},
			Nlris:    []*any.Any{nlri},
		})
	}
	return nlri, []*any.Any{a1, a2, a3}
}

//
// Build BGP routes for all static routes (configuration file)
func (r *anycastType) addRoutes(c *anycastConfig) error {
	for _, a := range c.Anycast.Routes {
		i, n, err := net.ParseCIDR(a.IP)
		if err == nil {
			cidrLength, _ := n.Mask.Size()

			if i.To4() != nil {
				// IPv4
				nlri, attr := createBGPRoute(n, config.Anycast.NextHopV4, a.LocalPref)

				r.routeTable = append(r.routeTable, &anycastRoute{
					ifName:        "static-v4",
					ifLabel:       a.Label,
					ipAddr:        n,
					ifDetail:      netlink.Addr{},
					LocalPref:     a.LocalPref,
					bgpNLRI:       nlri,
					bgpAttributes: attr,
				})

				log.WithFields(log.Fields{
					"interface": a.Label,
					"ip":        n.IP.String(),
					"cidr":      cidrLength,
					"localpref": a.LocalPref,
				}).Info("IPv4 route added")
			} else {
				// IPv6
				nlri, attr := createBGPRoute(n, config.Anycast.NextHopV6, a.LocalPref)

				r.routeTable = append(r.routeTable, &anycastRoute{
					ifName:        "static-v6",
					ifLabel:       a.Label,
					ipAddr:        n,
					ifDetail:      netlink.Addr{},
					LocalPref:     a.LocalPref,
					bgpNLRI:       nlri,
					bgpAttributes: attr,
				})

				log.WithFields(log.Fields{
					"interface": a.Label,
					"ip":        n.IP.String(),
					"cidr":      cidrLength,
					"localpref": a.LocalPref,
				}).Info("IPv6 route added")
			}
		} else {
			return err
		}
	}

	return nil
}

//
// Update BGP announcements to peers
func (r *anycastType) updateRoute() {
	var err error

	log.WithFields(log.Fields{
		"action": "route-update",
		"state":  r.state}).Info("Routing update")

	for _, a := range r.routeTable {
		var ipFamily *api.Family

		if a.ipAddr.IP.To4() != nil {
			ipFamily = &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_UNICAST}
		} else {
			ipFamily = &api.Family{Afi: api.Family_AFI_IP6, Safi: api.Family_SAFI_UNICAST}
		}

		if r.state {
			_, err = r.s.AddPath(context.Background(), &api.AddPathRequest{
				Path: &api.Path{
					Family: ipFamily,
					Nlri:   a.bgpNLRI,
					Pattrs: a.bgpAttributes,
				},
			})
		} else {
			err = r.s.DeletePath(context.Background(), &api.DeletePathRequest{
				Path: &api.Path{
					Family: ipFamily,
					Nlri:   a.bgpNLRI,
					Pattrs: a.bgpAttributes,
				},
			})
		}

		if err != nil {
			log.Error(err)
		}
	}
}
