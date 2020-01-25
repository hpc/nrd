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