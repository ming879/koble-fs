#!/bin/bash

buildah rm --all

set -e

wc=$(buildah from --dns none docker.io/library/debian:10)

#buildah run $wc update-alternatives --set iptables /usr/sbin/iptables-legacy

echo "Setting DEBIAN_FRONTEND to noninteractive"
buildah config --env DEBIAN_FRONTEND="noninteractive" $wc

echo "Running apt update"
buildah run $wc apt update --assume-yes

echo "Set debconf selections"
cat debconf-package-selections | buildah run $wc debconf-set-selections

echo "Apt install all packages"
PACKAGES_LIST=`cat packages-list | grep -v '#'`
buildah run $wc apt install --assume-yes ${PACKAGES_LIST}

cat ./packages-custom | buildah run $wc bash -

#buildah run $wc apt install --assume-yes --no-install-recommends wireguard-tools

echo "Add netkit user"
buildah run $wc useradd netkit -m -s /bin/bash -u 1000 -p $(openssl passwd -crypt netkit) -G sudo

echo "Set initial CMD"
buildah config --cmd "/sbin/init" $wc

echo "Copying Filesystem Tweaks"
buildah copy $wc filesystem-tweaks /

echo "Copying default homedirs"
buildah run $wc mkdir -p /root
buildah copy $wc HOME /root
buildah run $wc mkdir -p /home/netkit
buildah copy $wc HOME /home/netkit

echo "Enabling netkit systemd services"
buildah run $wc systemctl enable netkit-startup-phase1.service
buildah run $wc systemctl enable netkit-startup-phase2.service
buildah run $wc systemctl enable netkit-shutdown.service

# sort out ttys and auto-logon
buildah run $wc ln -s /lib/systemd/system/getty@.service /etc/systemd/system/getty.target.wants/getty@tty0.service
buildah run $wc systemctl mask getty-static
for i in {2..6}; do
    buildah run $wc systemctl mask getty@tty${i}.service
done

echo "Disable uneccessary services"
DISABLED_SERVICES=`cat disabled-services`
for SERVICE in $DISABLED_SERVICES; do
    buildah run $wc systemctl disable $SERVICE
done

echo "Commiting image"
buildah commit $wc netkit-deb-test

buildah rm $wc
