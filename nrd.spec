Name:           nrd
Version:        1.0
Release:        rc1%{?dist}
Summary:        nrd dynamically manages ECMP/MultiPath routes by listening for OSPF Hello packets

License:        BSD-3
URL:            https://gitlab.newmexicoconsortium.org/usrc/ngss/nrd
Source0:        nrd-1.0rc1.tar.gz

BuildRequires:  go, golang >= 1.13, golang-bin, golang-src
Requires: ethcfg >= 2.1

%description
Pronounced "nord" (which is the French for north), nrd creates routes based on a simple static configuration and OSPF hello packets.
This is intended to be run on HPC nodes to auto-configure their routes to be load-balanced and resilient to router failure.

%prep
%setup -q

%build
go build .


%install
mkdir -p %{buildroot}
install -D -s -m 0755 nrd %{buildroot}/sbin/nrd
install -D -m 0622 nrd.yml %{buildroot}/etc/nrd.yml
#install -m 0622 nrd.8.gz %{buildroot}/usr/share/man/man8/
install -m 0622 LICENSE %{buildroot}/usr/share/licenses/
install -D -m 0622 systemd/nrd.service %{buildroot}/usr/lib/systemd/system/nrd.service
install -D -m 0622 systemd/nrd.environment %{buildroot}/etc/sysconfig/nrd

%files
%defattr(-,root,root,-)
%doc LICENSE
/sbin/nrd
%config /etc/nrd.yml
%config /etc/sysconfig/nrd
#%config /etc/systemd/system/nrd.service
/usr/lib/systemd/system/nrd.service

%changelog

* Wed Jan 29 2020 J. Lowell Wofford <lowell@lanl.gov> 1.0-rc1
- Initial RPM build of nrd

