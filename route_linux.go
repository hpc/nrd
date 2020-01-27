package main

// +build linux

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/vishvananda/netlink"
)

var routes = map[string]*Route{}

// atomic counting of routes
// this allows us to create hooks based on how many are active
type routesCount int32

func (rc *routesCount) Up() {
	c := atomic.AddInt32((*int32)(rc), 1)
	n := len(routes)
	if conf.notify && !notifySent && int(c) == n {
		l.INFO("routes have initialized, sending sd_notify")
		// TODO: actually send sd_notify
		notifySent = true
	}
	l.INFO("there are %d/%d routes up", c, n)
}

func (rc *routesCount) Down() {
	c := atomic.AddInt32((*int32)(rc), -1)
	l.INFO("there are %d/%d routes up", c, len(routes))
}

var notifySent bool = false
var routesUp routesCount = 0

type Route struct {
	sync.Mutex
	r     *netlink.Route
	nhops map[string]*netlink.NexthopInfo
	up    bool
}

func NewRoute(dst *net.IPNet) (r *Route) {
	r = &Route{
		r: &netlink.Route{
			Dst:       dst,
			MultiPath: []*netlink.NexthopInfo{},
		},
		nhops: make(map[string]*netlink.NexthopInfo),
		up:    false,
	}
	return
}

// NOTE: does not lock, expects it's already held
func (r *Route) update() {
	nh := []*netlink.NexthopInfo{}
	for _, i := range r.nhops {
		nh = append(nh, i)
	}
	r.r.MultiPath = nh
	if r.up {
		if len(nh) == 0 {
			// down the route
			if err := netlink.RouteDel(r.r); err != nil {
				l.ERROR("failed to delete route: %v", err)
				return
			}
			r.up = false
			routesUp.Down()
			l.INFO("route %s is down", r.r.Dst.String())
		} else {
			if err := netlink.RouteReplace(r.r); err != nil {
				l.ERROR("failed to update route: %v", err)
				return
			}
			l.INFO("updated route %s", r.r.Dst.String())
		}
	} else if len(nh) > 0 {
		// route is currently down, bring it up
		if err := netlink.RouteAdd(r.r); err != nil {
			l.ERROR("failed to add route: %v", err)
			return
		}
		r.up = true
		routesUp.Up()
		l.INFO("route %s is down", r.r.Dst.String())
	}
}

func (r *Route) Add(gw net.IP) {
	r.Lock()
	defer r.Unlock()
	r.nhops[gw.String()] = &netlink.NexthopInfo{
		Gw: gw,
	}
	r.update()
}

func (r *Route) Del(gw net.IP) {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.nhops[gw.String()]; ok {
		delete(r.nhops, gw.String())
	}
	r.update()
}
