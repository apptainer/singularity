BootStrap: busybox
MirrorURL: https://www.busybox.net/downloads/binaries/1.26.1-defconfig-multiarch/busybox-x86_64

%post
    echo "Hello from inside the container"

%runscript
    echo "Running command: $*"
    exec "$@"
