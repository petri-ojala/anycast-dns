package main

import (
	"net"

	"github.com/miekg/dns"
)

//
// Query DNS
func queryDNS(queryName string, dnsType uint16) (*dns.Msg, error) {
        var dnsR *dns.Msg
        var err error

        c := dns.Client{
                Net:     "udp",
                Timeout: config.DNS.Timeout,
        }
        m := new(dns.Msg)
        m.SetQuestion(queryName, dnsType)
        m.RecursionDesired = true

        // Send query to the DNS
        dnsR, _, err = c.Exchange(m, net.JoinHostPort(config.DNS.ServerIP, config.DNS.ServerPort))
        return dnsR, err
}

//
// Check if DNS is responding
func checkDNS() bool {
        r, err := queryDNS(config.DNS.Query, dns.TypeSOA)
        if err != nil || r.Rcode != dns.RcodeSuccess {
                return false
        }
        return true
}

