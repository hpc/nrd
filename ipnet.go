package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// IPNet wraps net.IPNet this is used to unmarshal IPNets because net doesn't provide this function
type IPNet net.IPNet

// UnmarshalText decodes text of the form IP/MASK to an IPNET where MASK is either CIDR notation or dotted decimals
func (i *IPNet) UnmarshalText(text []byte) error {
	// split into net and mask
	p := strings.Split(string(text), "/")
	if len(p) != 2 {
		return fmt.Errorf("invalid IPNet: %s", text)
	}
	// parse net
	if err := i.IP.UnmarshalText([]byte(p[0])); err != nil {
		return err
	}
	// parse mask
	m := strings.Split(p[1], ".")
	switch len(m) {
	case 1:
		c, err := strconv.Atoi(m[0])
		if err != nil || c > 32 {
			return fmt.Errorf("invalid IPMask: %s", p[1])
		}
		i.Mask = net.CIDRMask(c, 32)
	case 4:
		var ip net.IP
		if err := ip.UnmarshalText([]byte(p[1])); err != nil {
			return fmt.Errorf("invalid IPMask: %s", p[1])
		}
		i.Mask = net.IPMask(ip.To4())
	default:
		return fmt.Errorf("invalid IPMask: %s", p[1])
	}
	return nil
}
