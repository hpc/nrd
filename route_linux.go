/* route_linux.go: provides (Linux specific) objects for managing netlink routes
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
	"net"
	"sync"
	"sync/atomic"

	"github.com/coreos/go-systemd/daemon"
	"github.com/vishvananda/netlink"
)

var routes = map[string]*Route{}

// atomic counting of routes
// this allows us to create hooks based on how many are active
type routesCount int32

func (rc *routesCount) Up() {
	c := atomic.AddInt32((*int32)(rc), 1)
	n := len(routes)
	if int(c) == n {
		notify()
	}
	l.INFO("there are %d/%d routes up", c, n)
}


func notify() {
	if conf.notify && !notifySent {
		l.INFO("routes have initialized, sending sd_notify")
		sent, err := daemon.SdNotify(false, daemon.SdNotifyReady)
		if err != nil {
			l.ERROR("failed to send sd_notify: %v", err)
		} else if !sent {
			l.WARN("notify was requested, but notification is not supported")
		} else {
			notifySent = true
		}
	}
}

func (rc *routesCount) Down() {
	c := atomic.AddInt32((*int32)(rc), -1)
	l.INFO("there are %d/%d routes up", c, len(routes))
}

var notifySent bool = false
var routesUp routesCount = 0

// A Route represents a kernel route with one or more nexthops and is used to update kernel routes
type Route struct {
	sync.Mutex
	r     *netlink.Route
	nhops map[string]*netlink.NexthopInfo
	up    bool
}

// NewRoute returns an initialized Route object
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
			notify()
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

// Add adds a route to the route table
func (r *Route) Add(gw net.IP) {
	r.Lock()
	defer r.Unlock()
	r.nhops[gw.String()] = &netlink.NexthopInfo{
		Gw: gw,
	}
	r.update()
}

// Del deletes a route from the route table
func (r *Route) Del(gw net.IP) {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.nhops[gw.String()]; ok {
		delete(r.nhops, gw.String())
	}
	r.update()
}

// SetUp forces the current state of the route to be up
// This effectively forces a RouteReplace instead of a RouteAdd
func (r *Route) SetUp() {
	r.up = true
}

// Exists checks to see if this route already exists in the route table
// It matches only on whether the Dst is the same as an existing route
func (r *Route) Exists() bool {
	match, err := netlink.RouteListFiltered(netlink.FAMILY_V4, r.r, netlink.RT_FILTER_DST)
	if err != nil {
		l.ERROR("error listing routes: %v", err)
		return true
	}
	if len(match) > 0 {
		return true
	}
	return false
}

// Cleanup removes the route.  This is used on exit by default.
func (r *Route) Cleanup() {
	if err := netlink.RouteDel(r.r); err != nil {
		l.ERROR("failed to delete route: %v", err)
		return
	}
}
