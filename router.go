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
	routes map[string]*Route
	dead   time.Duration
	timer  *time.Timer
	up     bool
	rObj   []netlink.Route
}

// NewRouter creates a new router.  New routers always start in down state
func NewRouter(ip net.IP, rs map[string]*Route, dead time.Duration) (r *Router) {
	r = &Router{
		ip:     ip,
		routes: routes,
		dead:   dead,
		timer:  &time.Timer{},
		up:     false,
		rObj:   []netlink.Route{},
	}
	return
}

// Up sets router up, starts dead timer
func (r *Router) Up() {
	r.Lock()
	defer r.Unlock()
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
		for _, route := range r.routes {
			route.Add(r.ip)
		}
	} else {
		l.DEBUG("dry run set, not adding route")
	}
	c := atomic.AddInt32(&routersUp, 1)
	l.INFO("there are %d routers up", c)
}

// Down sets router down
func (r *Router) Down() {
	r.Lock()
	defer r.Unlock()
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
		for _, route := range r.routes {
			route.Del(r.ip)
		}
	} else {
		l.DEBUG("dry run set, not removing route")
	}
	c := atomic.AddInt32(&routersUp, -1)
	l.INFO("there are %d routers up", c)
}

// Dead sets router dead. This also calls Down
func (r *Router) Dead() {
	l.WARN("router %s hit dead state", r.ip.String())
	r.Down()
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
