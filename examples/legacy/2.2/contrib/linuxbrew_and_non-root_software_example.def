BootStrap: debootstrap
OSVersion: trusty
MirrorURL: http://us.archive.ubuntu.com/ubuntu/

%runscript
	# print out software versions installed by linuxbrew
	find /Software/brew/Cellar -maxdepth 2 -print | sed 's|/Software/brew/Cellar||g' | sed 's|^/||' | grep "/" | sed 's|/|\t|' | sort | awk '{print $1, $2, "Homebrew"}' | column -t | sort -u --ignore-case

%post
	sed -i 's/$/ universe/' /etc/apt/sources.list
	locale-gen "en_US.UTF-8"
	dpkg-reconfigure locales
	export LANGUAGE="en_US.UTF-8"
	echo 'LANGUAGE="en_US.UTF-8"' >> /etc/default/locale
	echo 'LC_ALL="en_US.UTF-8"' >> /etc/default/locale
	mkdir /Software
	chmod 777 /tmp
	chmod +t /tmp
	chmod 777 /Software
	apt-get update
	apt-get install -y apt-transport-https build-essential cmake curl libsm6 libxrender1 libfontconfig1 wget vim git unzip python-setuptools ruby
	apt-get clean
	useradd -m singularity
	su -c 'cd /Software && git clone https://github.com/Linuxbrew/brew.git' singularity
	su -c '/Software/brew/bin/brew install bsdmainutils parallel util-linux' singularity
	su -c '/Software/brew/bin/brew tap homebrew/science' singularity
	su -c '/Software/brew/bin/brew install art bwa samtools' singularity
	sed -i 's|PATH=$PATH:/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin|PATH="/Software/brew/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"|' /environment
