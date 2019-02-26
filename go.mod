module github.com/sylabs/singularity

go 1.12

require (
	github.com/Microsoft/go-winio v0.4.7
	github.com/Sirupsen/logrus v0.0.0-00010101000000-000000000000 // indirect
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973
	github.com/blang/semver v3.5.1+incompatible
	github.com/containerd/cgroups v0.0.0-20181208203134-65ce98b3dfeb
	github.com/containerd/continuity v0.0.0-20180612233548-246e49050efd
	github.com/containernetworking/cni v0.6.0
	github.com/containernetworking/plugins v0.0.0-20180606151004-2b8b1ac0af45
	github.com/containers/image v0.0.0-20180612162315-2e4f799f5eba
	github.com/containers/storage v0.0.0-20180604200230-88d80428f9b1
	github.com/coreos/go-iptables v0.3.0
	github.com/coreos/go-systemd v0.0.0-20180511133405-39ca1b05acc7
	github.com/cpuguy83/go-md2man v1.0.8
	github.com/d2g/dhcp4 v0.0.0-20170904100407-a1d1b6c41b1c
	github.com/d2g/dhcp4client v0.0.0-20180611075603-e61299896203
	github.com/docker/distribution v0.0.0-20180611183926-749f6afb4572
	github.com/docker/docker v0.0.0-20180522102801-da99009bbb11
	github.com/docker/docker-credential-helpers v0.6.0
	github.com/docker/go-connections v0.3.0
	github.com/docker/go-metrics v0.0.0-20180209012529-399ea8c73916
	github.com/docker/go-units v0.3.3
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7
	github.com/ghodss/yaml v1.0.0
	github.com/globalsign/mgo v0.0.0-20180615134936-113d3961e731
	github.com/godbus/dbus v4.1.0+incompatible
	github.com/gogo/protobuf v1.0.0
	github.com/golang/protobuf v1.1.0
	github.com/gorilla/context v1.1.1
	github.com/gorilla/mux v1.6.2
	github.com/gorilla/websocket v1.2.0
	github.com/hashicorp/errwrap v0.0.0-20141028054710-7554cd9344ce
	github.com/hashicorp/go-multierror v0.0.0-20171204182908-b7773ae21874
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/j-keck/arping v0.0.0-20160618110441-2cf9dc699c56
	github.com/kr/pty v1.1.3
	github.com/kubernetes-sigs/cri-o v0.0.0-20180917213123-8afc34092907
	github.com/magiconair/properties v1.8.0
	github.com/mattn/go-runewidth v0.0.2
	github.com/mattn/go-shellwords v1.0.3
	github.com/matttproud/golang_protobuf_extensions v1.0.1
	github.com/mtrmac/gpgme v0.0.0-20170102180018-b2432428689c
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/opencontainers/image-spec v0.0.0-20180411145040-e562b0440392
	github.com/opencontainers/image-tools v0.3.0
	github.com/opencontainers/runc v0.1.1
	github.com/opencontainers/runtime-spec v0.0.0-20180913141938-5806c3563733
	github.com/opencontainers/runtime-tools v0.6.0
	github.com/opencontainers/selinux v1.0.0-rc1
	github.com/pelletier/go-toml v1.2.0
	github.com/pkg/errors v0.8.0
	github.com/pquerna/ffjson v0.0.0-20171002144729-d49c2bc1aa13
	github.com/prometheus/client_golang v0.0.0-20180607123607-faf4ec335fe0
	github.com/prometheus/client_model v0.0.0-20171117100541-99fa1f4be8e5
	github.com/prometheus/common v0.0.0-20180518154759-7600349dcfe1
	github.com/prometheus/procfs v0.0.0-20180612222113-7d6f385de8be
	github.com/russross/blackfriday v1.5.1
	github.com/safchain/ethtool v0.0.0-20180504150752-6e3f4faa84e1
	github.com/satori/go.uuid v1.2.0
	github.com/seccomp/libseccomp-golang v0.9.0
	github.com/sirupsen/logrus v1.3.0
	github.com/spf13/cobra v0.0.0-20180531180338-1e58aa3361fd
	github.com/spf13/pflag v1.0.1
	github.com/sylabs/json-resp v0.1.0
	github.com/sylabs/sif v1.0.2
	github.com/syndtr/gocapability v0.0.0-20180223013746-33e07d32887e
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20171111001504-be1fbeda1936
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415
	github.com/xeipuuv/gojsonschema v0.0.0-20180618132009-1d523034197f
	go4.org v0.0.0-20180417224846-9599cf28b011
	golang.org/x/crypto v0.0.0-20190225124518-7f87c0fbb88b
	golang.org/x/net v0.0.0-20180611182652-db08ff08e862
	golang.org/x/sys v0.0.0-20180905080454-ebe1bf3edb33
	gopkg.in/cheggaaa/pb.v1 v1.0.25
	gopkg.in/yaml.v2 v2.2.1
)

replace (
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.0.5
	github.com/opencontainers/image-tools => github.com/sylabs/image-tools v0.0.0-20181006203805-2814f4980568
	golang.org/x/crypto => github.com/sylabs/golang-x-crypto v0.0.0-20181006204705-4bce89e8e9a9
)
