#!/bin/sh

if [ -f "/etc/debian_version" ]; then
 sudo apt-get update && sudo apt-get install -y build-essential libssl-dev uuid-dev libgpgme11-dev squashfs-tools libseccomp-dev wget pkg-config git cryptsetup
fi
if [ "$(grep -Ei 'fedora|redhat' /etc/*release)" ]; then
 sudo dnf update && sudo dnf install -y squashfs-tools wget pkg-config git cryptsetup gcc-go golang-bin
fi

rm -r /usr/local/go
wget https://dl.google.com/go/go1.15.1.linux-amd64.tar.gz #https://golang.org/doc/install
sudo tar -C /usr/local -xzf go1.15.1.linux-amd64.tar.gz
echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc && source ~/.bashrc

wget https://raw.githubusercontent.com/hpcng/singularity/master/CHANGELOG.md
pid=$(grep "^# v" ./CHANGELOG.md)
export VERSION=${pid:3:5}

wget https://github.com/sylabs/singularity/releases/download/v${VERSION}/singularity-${VERSION}.tar.gz
tar -xzf singularity-${VERSION}.tar.gz
cd singularity/
./mconfig
make -C ./builddir
sudo make -C ./builddir install
which singularity
singularity --version
cd ..
