package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

var l *logger

// init config struct, set defaults
var conf = &struct {
	cfgFile, ifaceName, mcastAddr string
	logLevel                      logLevel
	mcast, notify, up             bool
}{
	cfgFile:   "nrd.yml",
	logLevel:  DEBUG,
	ifaceName: "eth0",
	mcast:     false,
	notify:    false,
	mcastAddr: "224.0.0.5",
	up:        false,
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
	flag.BoolVar(&conf.mcast, "mcast", conf.mcast, "join multicast group")
	flag.StringVar(&conf.ifaceName, "iface", conf.ifaceName, "interface to listen on")
	flag.StringVar(&conf.cfgFile, "conf", conf.cfgFile, "configuration file to use")
	flag.BoolVar(&conf.notify, "notify", conf.notify, "send sd_notify messages")
	flag.BoolVar(&conf.up, "up", conf.up, "set startup state of routes to up")
	lvl := flag.Uint("log", uint(conf.logLevel), "set the log level [0-4]")
	flag.Parse()
	conf.logLevel = logLevel(*lvl)

	// create logger
	l = NewLogger(os.Stdout, conf.logLevel)
	l.INFO("Starting NRD")

	l.DEBUG("conf = %+v", *conf)

	// read config
	l.INFO("Reading config file at: %s", conf.cfgFile)

	data, err := ioutil.ReadFile(conf.cfgFile)
	if err != nil {
		l.FATAL("couldn't read cfg file: %v", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		l.FATAL("failed to parse config file: %v", err)
	}
	l.DEBUG("cfgFile: %+v", *cfg)
	// TODO: should probably do some sanity checking?

	// create & populate router objects
	for _, rip := range cfg.Routers {
		routers[rip.String()] = NewRouter(rip, cfg.Routes, cfg.Dead)
		l.INFO("added router %s", rip.String())
		if conf.up {
			routers[rip.String()].Up()
		}
	}

	// initialize packet listener

	// join multicast group

	// main listener loop
	for {

	}
}
