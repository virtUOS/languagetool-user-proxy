#!/bin/bash

set -eux

dnf install -y rpmdevtools
rpmdev-setuptree

echo "%appversion ${2}" >>~/.rpmmacros
cp "/rpmbuild/${1}" ~/rpmbuild/SOURCES/languagetool-user-proxy
cp /rpmbuild/.env.example ~/rpmbuild/SOURCES/
cp /rpmbuild/.rpm/languagetool-user-proxy.service ~/rpmbuild/SOURCES/
cp /rpmbuild/.rpm/languagetool-user-proxy.spec ~/rpmbuild/SPECS/
cp /rpmbuild/LICENSE ~/rpmbuild/BUILD/
cp /rpmbuild/README.md ~/rpmbuild/BUILD/

rpmbuild -ba --target "${3}" ~/rpmbuild/SPECS/languagetool-user-proxy.spec

cp ~/rpmbuild/RPMS/${3}/languagetool-user-proxy*.rpm /rpmbuild/
