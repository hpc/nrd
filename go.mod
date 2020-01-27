module github.com/hpc/nrd

go 1.13

replace github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0

require (
	github.com/coreos/go-systemd v0.0.0-00010101000000-000000000000
	github.com/google/gopacket v1.1.17
	github.com/vishvananda/netlink v1.1.0
	golang.org/x/net v0.0.0-20200114155413-6afb5195e5aa
	gopkg.in/yaml.v2 v2.2.8
)
