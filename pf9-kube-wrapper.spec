# Copyright (c) 2014 Platform9 Systems Inc.
# Spec file for creating wrapper-rpm for the qbert kubernetes distro.

Name:		pf9-kube-wrapper
Version:	%{_version}
Release:	pmk.__BUILDNUM__.__GITHASH__
Summary:	Platform9 pf9-kube wrapper

License:        Commercial
Group:          Platform9
URL:		http://wwww.platform9.net
Provides:      pf9-kube-wrapper
%description
Wraps kube RPM and DEB in another rpm

%prep

%build
%install
SRC_DIR=%_src_dir
INSTALL_DIR=$RPM_BUILD_ROOT/opt/pf9/www/private/pf9-kube/%{_version}-pmk.__BUILDNUM__
mkdir -p $INSTALL_DIR
cp $SRC_DIR/*.rpm $INSTALL_DIR
cp $SRC_DIR/*.deb $INSTALL_DIR
cp $SRC_DIR/role.json $INSTALL_DIR
cp $SRC_DIR/addons.json $INSTALL_DIR
cp $SRC_DIR/metadata.json $INSTALL_DIR

%clean
rm -rf %{buildroot}

%post
# pf9-resmgr requires restart to refresh the active pf9-kube role
/usr/bin/mkdir -p /opt/pf9/qbert/supportedRoleVersions
/usr/bin/touch /opt/pf9/qbert/supportedRoleVersions/%{_version}-pmk.__BUILDNUM__
/usr/bin/systemctl restart pf9-qbert >/dev/null 2>&1 || true

%files
%defattr(-,root,root,-)
/opt/pf9
