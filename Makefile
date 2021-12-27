default: uml-image

.PHONY: clean
clean:
	rm -rf build

archive: uml-image
	tar -C build cvf netkit-fs.tar.xz netkit-fs

uml-image: netkit-image ./uml.sh
	buildah unshare ./uml.sh

netkit-image: $(find ./filesystem-tweaks) ./packages-list ./packages-custom ./disabled-services ./debconf-package-selections ./build.sh
	./build.sh
