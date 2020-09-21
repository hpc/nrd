Name:           nrd
Version:        1.0
Release:        1%{?dist}
Summary:        nrd dynamically manages ECMP/MultiPath routes by listening for OSPF Hello packets
Group:          Applications/System
License:        BSD-3
URL:            https://gitlab.newmexicoconsortium.org/usrc/ngss/nrd
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  go, golang >= 1.12, golang-bin, golang-src
Requires: NetworkManager

%define  debug_package %{nil}

%description
Pronounced "nord" (which is the French for north), nrd creates routes based on a simple static configuration and OSPF hello packets.
This is intended to be run on HPC nodes to auto-configure their routes to be load-balanced and resilient to router failure.

%prep
%setup -q

%build
go build .
gzip nrd.8


%install
mkdir -p %{buildroot}
install -D -s -m 0755 nrd %{buildroot}%{_sbindir}/nrd
install -D -m 0644 nrd.yml %{buildroot}%{_sysconfdir}/nrd.yml
install -D -m 0644 nrd.8.gz %{buildroot}%{_mandir}/man8/nrd.8.gz
install -D -m 0644 systemd/nrd.service %{buildroot}%{_unitdir}/nrd.service
install -D -m 0644 systemd/nrd.environment %{buildroot}%{_sysconfdir}/sysconfig/nrd
install -D -m 0755 wait-for-ifs.py %{buildroot}%{_libexecdir}/wait-for-ifs
install -D -m 0644 systemd/wait-for-ifs.service %{buildroot}%{_unitdir}/wait-for-ifs.service
install -D -m 0644 systemd/wait-for-ifs.environment %{buildroot}%{_sysconfdir}/sysconfig/wait-for-ifs

%files
%defattr(-,root,root)
%license LICENSE
%doc %{_mandir}/man8/nrd.8.gz
%{_sbindir}/nrd
%config(noreplace) %{_sysconfdir}/nrd.yml
%config(noreplace) %{_sysconfdir}/sysconfig/nrd
%{_unitdir}/nrd.service
%{_libexecdir}/wait-for-ifs
%{_unitdir}/wait-for-fs.service
%config(noreplace) %{_sysconfdir}/sysconfig/wait-for-ifs

%changelog

* Mon Sep 21 2020 J. Lowell Wofford <lowell@lanl.gov> 1.0-2
- Add wait-for-ifs script/service
* Fri Aug 28 2020 J. Lowell Wofford <lowell@lanl.gov> 1.0-1
- Disable debug_package since the debuginfo build fails on RHEL8/CentOS8.
* Tue Apr 07 2020 J. Lowell Wofford <lowell@lanl.gov> 1.0-0
- Rev to 1.0
* Wed Jan 29 2020 J. Lowell Wofford <lowell@lanl.gov> 1.0-rc1
- Initial RPM build of nrd
