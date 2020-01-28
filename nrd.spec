Name:           nrd
Version:        1.0
Release:        rc1%{?dist}
Summary:        nrd dynamically manages ECMP/MultiPath routes by listening for OSPF Hello packets

License:        BSD-3
URL:            https://gitlab.newmexicoconsortium.org/usrc/ngss/nrd
Source0:        nrd-1.0rc1.tar.gz

BuildRequires:  go golang golang-bin golang-src
Requires:       

%description
Pronounced "nord" (which is the French for north), nrd creates routes based on a simple static configuration and OSPF hello packets.
This is intended to be run on HPC nodes to auto-configure their routes to be load-balanced and resilient to router failure.

%prep
%setup -q

%build
/bin/sh build.sh

%install
mkdir -p %{buildroot}
install -s -m 0755 nrd %{buildroot}/usr/libexec/
install -m 0622 nrd.yaml %{buildroot}/etc/sysconfig/
install -m 0622 nrd.8.gz %{buildroot}/usr/share/man/man8/
install -m 0622 LICENSE %{buildroot}/usr/share/licenses/
install -m 0622 systemd/nrd.service %{buildroot}/etc/systemd/system
install -m 0622 systemd/nrd.service %{buildroot}/usr/lib/systemd/system
install -m 0622 systemd/nrd.environment %{buildroot}/etc/sysconfig/

%files
%defattr(-,root,root,-)
%doc LICENSE
/usr/libexec/nrd
%config /etc/sysconfig/nrd.yaml
%config /etc/systemd/system/nrd.service
/usr/lib/systemd/system/nrd.service

%changelog

* Date Name <email> 1.0.-rc1
- Initial RPM build of nrd
