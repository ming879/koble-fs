#!/bin/bash

mkdir -p build
wc=$(buildah from netkit-deb-test)

# Set CAD action to poweroff instead of reboot
# this allows us to use uml_mconsole to cleanly shutdown a UML instance
buildah run $wc ln -s /lib/systemd/system/poweroff.target /etc/systemd/system/ctrl-alt-del.target

# Install udev
buildah run $wc apt update --assume-yes
buildah run $wc apt install udev


dd if=/dev/zero of=build/netkit-fs bs=1 count=1 seek=4G
mkfs.ext4 build/netkit-fs

# copy from container to bootstrap-fs
mntsrc=$(buildah mount $wc)
echo "mounted buildah container at $mntsrc"
mntdst="build/mountimage"
mkdir $mntdst
ls -lha build/netkit-fs
fuse-ext2 build/netkit-fs build/mountimage -o rw+

echo "Copying from $mntsrc $mntdst"
cp -r ${mntsrc}/* $mntdst

umount build/mountimage
buildah umount $wc
