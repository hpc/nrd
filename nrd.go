/* nrd.go: entry point for Neighborless Route Detection (nrd)
 *
 * Authors: J. Lowell Wofford <lowell@lanl.gov> & Brett Holman <bholman@lanl.gov>
 *
 * This software is open source software available under the BSD-3 license.
 * Copyright (c) 2020, Triad National Security, LLC
 * See LICENSE file for details.
 */

// +build linux

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/net/ipv4"
	"gopkg.in/yaml.v2"
)

var l *logger

const version = "1.0rc1"

const protocol = "ip4:89"

// init config struct, set defaults
var conf = &struct {
	cfgFile, ifaceName, mcastAddr           string
	logLevel                                logLevel
	notify, up, nojoin, dry, force, noclean bool
}{
	cfgFile:   "/etc/nrd.yml",
	logLevel:  INFO,
	ifaceName: "eth0",
	notify:    false,
	mcastAddr: "224.0.0.5",
	up:        false,
	nojoin:    false,
	dry:       false,
	force:     false,
	noclean:   false,
}

// format of config file
type cfgFile struct {
	Dead    time.Duration
	Routes  []IPNet
	Routers []net.IP
}

var cfg = &cfgFile{}

func sigHandle(c <-chan os.Signal) {
	for {
		switch s := <-c; s {
		case syscall.SIGTERM:
			fallthrough
		case os.Interrupt:
			if !conf.noclean {
				l.INFO("exiting. cleaning up managed routes.")
				for _, r := range routes {
					r.Cleanup()
				}
			} else {
				l.INFO("exiting. option noclean specified, leaving managed routes.")
			}
			os.Exit(0)
		}
	}
}

func main() {

	// parse flags
	flag.StringVar(&conf.ifaceName, "iface", conf.ifaceName, "interface to listen on")
	flag.StringVar(&conf.cfgFile, "conf", conf.cfgFile, "configuration file to use")
	flag.BoolVar(&conf.notify, "notify", conf.notify, "send sd_notify messages")
	flag.BoolVar(&conf.up, "up", conf.up, "set startup state of routes to up")
	flag.BoolVar(&conf.nojoin, "nojoin", conf.nojoin, "don't join multicast (assume it's already joined)")
	flag.BoolVar(&conf.dry, "dry", conf.dry, "dryrun, don't actually set routes")
	flag.BoolVar(&conf.force, "force", conf.dry, "force nrd to control routes even if they already exist")
	flag.BoolVar(&conf.noclean, "noclean", conf.noclean, "don't cleanup managed routes on exit")
	pVersion := flag.Bool("version", false, "output version information and exit")
	lvl := flag.Uint("log", uint(conf.logLevel), "set the log level [0-3]")
	flag.Parse()
	conf.logLevel = logLevel(*lvl)

	if *pVersion {
		fmt.Printf(`
nrd %s

This software is open source software available under the BSD-3 license.
Copyright (c) 2020, Triad National Security, LLC
See LICENSE file for details.

Written by J. Lowell Wofford and Brett Holman.
`, version)
		os.Exit(0)
	}

	// create logger
	l = NewLogger(os.Stdout, conf.logLevel)
	l.INFO("starting NRD")

	// check if root
	if os.Geteuid() != 0 {
		l.FATAL("must be run with root privelege")
	}

	// trap interrupts to perform cleanup on exit
	sigChan := make(chan os.Signal)
	go sigHandle(sigChan)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	l.DEBUG("conf = %+v", *conf)

	// read config
	l.INFO("reading config file at: %s", conf.cfgFile)

	data, err := ioutil.ReadFile(conf.cfgFile)
	if err != nil {
		l.FATAL("couldn't read cfg file: %v", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		l.FATAL("failed to parse config file: %v", err)
	}
	l.DEBUG("cfgFile: %+v", *cfg)

	// get interface
	iface, err := net.InterfaceByName(conf.ifaceName)
	if err != nil {
		l.FATAL("failed to find interface %s: %v", conf.ifaceName, err)
	}
	l.INFO("using interface %s", conf.ifaceName)

	// get assicated ip addrs
	// we select the first IPv4 addr
	var ifaddr net.IP
	ifaddrs, err := iface.Addrs()
	if err != nil {
		l.FATAL("couldn't get interface addrs: %v", err)
	}
	for _, a := range ifaddrs {
		ip := a.(*net.IPNet).IP.To4()
		if ip != nil {
			ifaddr = ip
			break
		}
	}
	if ifaddr == nil {
		l.FATAL("interface %s has no IPv4 addresses", conf.ifaceName)
	}

	// create & populate route objects
	for _, rn := range cfg.Routes {
		n := net.IPNet(rn)
		nr := NewRoute(&n)
		if nr.Exists() {
			if conf.force {
				nr.SetUp()
				l.WARN("route %s exists but -force specified.  Route will be replaced.", n.String())
			} else {
				l.WARN("route %s exists, dropping from route list.  Use -force to take over existing routes.", n.String())
				continue
			}
		}
		routes[n.String()] = nr
		l.INFO("managing route %s", n.String())
	}

	// create & populate router objects
	for _, rip := range cfg.Routers {
		routers[rip.String()] = NewRouter(rip, routes, cfg.Dead)
		l.INFO("added router %s", rip.String())
		if conf.up {
			routers[rip.String()].Up()
		}
	}

	// initialize packet listener
	list, err := net.ListenPacket(protocol, conf.mcastAddr)
	defer list.Close()

	conn := ipv4.NewPacketConn(list)

	// join multicast group
	if !conf.nojoin {
		if err := conn.JoinGroup(iface, &net.UDPAddr{IP: net.ParseIP(conf.mcastAddr)}); err != nil {
			l.FATAL("failed to join multicast group: %v", err)
		}
		defer conn.LeaveGroup(iface, &net.UDPAddr{IP: net.ParseIP(conf.mcastAddr)})
		if err := conn.SetControlMessage(ipv4.FlagDst, true); err != nil {
			l.FATAL("failed to set message control: %v", err)
		}
		l.INFO("joined multicast group: %s", conf.mcastAddr)
	}

	// init packet decoder
	var p layers.OSPFv2
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeOSPF, &p)
	decoded := []gopacket.LayerType{}

	// make a buf the size of MTU
	buf := make([]byte, iface.MTU)

	// main listener loop
	for {
		n, _, saddr, err := conn.ReadFrom(buf)
		if err != nil {
			// this shouldn't be a failure condition, but it is an error
			l.ERROR("error reading packet: %v", err)
			continue
		}
		l.DEBUG("raw packet: %x", buf[:n])
		if err := parser.DecodeLayers(buf[:n], &decoded); err != nil {
			l.DEBUG("error decoding packet: %v", err)
			continue
		}
		for _, pack := range decoded {
			if pack != layers.LayerTypeOSPF {
				l.WARN("decoded non-OSPF packet")
				continue
			}
			l.DEBUG("received OSPF type: %s", p.Type.String())
			switch p.Type {
			case layers.OSPFHello:
				if r, ok := routers[saddr.(*net.IPAddr).IP.String()]; ok {
					go r.Hello()
				} else {
					l.WARN("got HELLO from unknown router: %s", saddr.(*net.IPAddr).IP.String())
				}
			default:
				l.DEBUG("unhandled OSPF type")
			}
		}
	}
}
