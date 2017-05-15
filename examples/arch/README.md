# Arch for Singularity

This bootstrap spec will generate an arch linux distribution using Singularity 2.3 (current development branch). Note that you can also just bootstrap a Docker image:


If you want to move forward with the raw, old school, jeans and hard toes bootstrap, here is what to do. I work on an Ubuntu machine, so I had to use a Docker Arch Linux image to do this. This first part you should do on your local machine (if not arch linux) is to use Docker to interactively work in an Arch Linux image. If you don't want to copy paste the build spec file, you can use `--volume` to mount a directory from your host to a folder in the image (I would recommend `/tmp` or similar). Here we run the docker image:


```bash
docker run -it  --privileged pritunl/archlinux bash
```

```bash
pacman -S -y git autoconf libtool automake gcc python make sudo vim arch-install-scripts wget
git clone https://github.com/singularityware/singularity
cd singularity
git checkout -b development
git pull origin development
./autogen.sh
./configure --prefix=/usr/local
```

You can add the [Singularity](Singularity) build spec here, or cd to where it is if you have mounted a volume.

```bash
cd /tmp
singularity create arch.img
sudo singularity bootstrap arch.img Singularity
```

That should do the trick!
