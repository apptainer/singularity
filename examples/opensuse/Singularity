BootStrap: zypper
OSVersion: 42.2
MirrorURL: http://download.opensuse.org/distribution/leap/%{OSVERSION}/repo/oss/
Include: zypper

%runscript
    echo "This is what happens when you run the container..."


%post
    echo "Hello from inside the container"
    zypper -n install bc
