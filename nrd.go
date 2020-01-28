package main

import (
	"flag"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/net/ipv4"
	"gopkg.in/yaml.v2"
)

var l *logger

const protocol = "ip4:89"

// init config struct, set defaults
var conf = &struct {
	cfgFile, ifaceName, mcastAddr  string
	logLevel                       logLevel
	notify, up, nojoin, dry, force bool
}{
	cfgFile:   "nrd.yml",
	logLevel:  INFO,
	ifaceName: "eth0",
	notify:    false,
	mcastAddr: "224.0.0.5",
	up:        false,
	nojoin:    false,
	dry:       false,
	force:     false,
}

// format of config file
type cfgFile struct {
	Dead    time.Duration
	Routes  []IPNet
	Routers []net.IP
}

var cfg = &cfgFile{}

func main() {

	// parse flags
	flag.StringVar(&conf.ifaceName, "iface", conf.ifaceName, "interface to listen on")
	flag.StringVar(&conf.cfgFile, "conf", conf.cfgFile, "configuration file to use")
	flag.BoolVar(&conf.notify, "notify", conf.notify, "send sd_notify messages")
	flag.BoolVar(&conf.up, "up", conf.up, "set startup state of routes to up")
	flag.BoolVar(&conf.nojoin, "nojoin", conf.nojoin, "don't join multicast (assume it's already joined)")
	flag.BoolVar(&conf.dry, "dry", conf.dry, "dryrun, don't actually set routes")
	flag.BoolVar(&conf.force, "force", conf.dry, "force nrd to control routes even if they already exist")
	lvl := flag.Uint("log", uint(conf.logLevel), "set the log level [0-3]")
	flag.Parse()
	conf.logLevel = logLevel(*lvl)

	// create logger
	l = NewLogger(os.Stdout, conf.logLevel)
	l.INFO("starting NRD")

	// check if root
	if os.Geteuid() != 0 {
		l.FATAL("must be run with root privelege")
	}

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
