package main

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vishvananda/netlink"
)

const (
	metric = 0
)

var routers = map[string]*Router{}
var routersUp int32 = 0

type Router struct {
	sync.Mutex
	ip     net.IP
	routes []IPNet
	dead   time.Duration
	timer  *time.Timer
	up     bool
	rObj   []netlink.Route
}

// NewRouter creates a new router.  New routers always start in down state
func NewRouter(ip net.IP, routes []IPNet, dead time.Duration) (r *Router) {
	iface, e := netlink.LinkByName(conf.ifaceName)
	if e != nil {
		l.FATAL("interface %v not found: %v", conf.ifaceName, e)
	}

	r = &Router{
		ip:     ip,
		routes: routes,
		dead:   dead,
		timer:  &time.Timer{},
		up:     false,
		rObj:   []netlink.Route{},
	}
	for _, v := range r.routes {
		dst := net.IPNet(v)
		route := netlink.Route{
			LinkIndex: iface.Attrs().Index,
			Gw:        r.ip,
			Dst:       &dst,
			Priority:  metric,
		}
		r.rObj = append(r.rObj, route)
	}
	return
}

// Up sets router up, starts dead timer
func (r *Router) Up() {
	r.Lock()
	if r.up == true {
		// already up
		l.DEBUG("router.Up called, but this route is already up")
		return
	}
	l.INFO("router %s is now up", r.ip.String())
	r.up = true
	r.timer = time.AfterFunc(r.dead, r.Dead)
	// set route
	if !conf.dry {
		for _, route := range r.rObj {
			if err := netlink.RouteAdd(&route); err != nil {
				l.ERROR("couldn't add route: %v", err)
			}
		}
	} else {
		l.DEBUG("dry run set, not adding route")
	}
	r.Unlock()
	c := atomic.AddInt32(&routersUp, 1)
	l.INFO("there are %d routers up", c)
}

// Down sets router down
func (r *Router) Down() {
	r.Lock()
	if r.up == false {
		// already down
		l.DEBUG("router.Down called, but this route is already down")
		return
	}
	l.INFO("router %s is now down", r.ip.String())
	r.timer.Stop()
	r.up = false
	// unset route
	if !conf.dry {
		for _, route := range r.rObj {
			if err := netlink.RouteDel(&route); err != nil {
				l.ERROR("couldn't add route: %v", err)
			}
		}
	} else {
		l.DEBUG("dry run set, not removing route")
	}
	r.Unlock()
	c := atomic.AddInt32(&routersUp, -1)
	l.INFO("there are %d routers up", c)
}

// Dead sets router dead. This also calls Down
func (r *Router) Dead() {
	l.WARN("router %s hit dead state", r.ip.String())
	r.Down()
}

// IsUp Checks if router is up
func (r *Router) IsUp() bool {
	return r.up // could technically be racey
}

// Hello reports a router hello
func (r *Router) Hello() {
	l.DEBUG("received a HELLO for %s", r.ip.String())
	r.Lock()
	if r.up {
		// just reset the timer
		r.timer.Reset(r.dead)
		r.Unlock()
	} else {
		// bring interface up
		l.INFO("router returned to service %s", r.ip.String())
		r.Unlock()
		r.Up()
	}
}
