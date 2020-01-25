# About

NRD = Neighborless Route Detection

The NRD process can manage ECMP/MultiPath routes on a network by listening to OSPF hello packets.

It takes YAML formatted config file (see `nrd.yml`) for an example.

It takes a number of arguments, which can be seen by running `nrd -h`.

NRD is currently only supported on Linux.

NRD requires Go version 1.13 or greater.

To build on a Linux system:

```console
$ go build .
```

To cross compile from another system or architecture, specify `GOOS` and `GOARCH`, e.g.:

```console
$ GOOS=linux GOARCH=arm64 go build .
```

Here's some example output of NRD in action with two routers on the network and some induced instability:

```console
[vagrant@leaf1 ~]$ sudo ./nrd -iface eth1 -log 2
2020/01/25 04:30:26 INFO:starting NRD
2020/01/25 04:30:26 INFO:reading config file at: nrd.yml
2020/01/25 04:30:26 INFO:using interface eth1
2020/01/25 04:30:26 INFO:managing route 192.168.57.0/24
2020/01/25 04:30:26 INFO:added router 192.168.56.10
2020/01/25 04:30:26 INFO:added router 192.168.56.11
2020/01/25 04:30:26 INFO:joined multicast group: 224.0.0.5
2020/01/25 04:30:32 INFO:router returned to service 192.168.56.10
2020/01/25 04:30:32 INFO:router 192.168.56.10 is now up
2020/01/25 04:30:32 INFO:route 192.168.57.0/24 is down
2020/01/25 04:30:32 INFO:there are 1 routers up
2020/01/25 04:30:32 INFO:router returned to service 192.168.56.11
2020/01/25 04:30:32 INFO:router 192.168.56.11 is now up
2020/01/25 04:30:32 INFO:updated route 192.168.57.0/24
2020/01/25 04:30:32 INFO:there are 2 routers up
```