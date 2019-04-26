module github.com/sylabs/singularity

go 1.11

require (
	github.com/Microsoft/go-winio v0.4.7 // indirect
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f // indirect
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/containerd/cgroups v0.0.0-20181208203134-65ce98b3dfeb
	github.com/containerd/continuity v0.0.0-20180612233548-246e49050efd // indirect
	github.com/containernetworking/cni v0.6.0
	github.com/containernetworking/plugins v0.0.0-20180606151004-2b8b1ac0af45
	github.com/containers/image v0.0.0-20180612162315-2e4f799f5eba
	github.com/containers/storage v0.0.0-20180604200230-88d80428f9b1 // indirect
	github.com/coreos/go-iptables v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20180511133405-39ca1b05acc7 // indirect
	github.com/d2g/dhcp4 v0.0.0-20170904100407-a1d1b6c41b1c // indirect
	github.com/d2g/dhcp4client v0.0.0-20180611075603-e61299896203 // indirect
	github.com/d2g/dhcp4server v0.0.0-20181031114812-7d4a0a7f59a5 // indirect
	github.com/d2g/hardwareaddr v0.0.0-20190221164911-e7d9fbe030e4 // indirect
	github.com/docker/distribution v0.0.0-20180611183926-749f6afb4572 // indirect
	github.com/docker/docker v0.0.0-20180522102801-da99009bbb11 // indirect
	github.com/docker/docker-credential-helpers v0.6.0 // indirect
	github.com/docker/go-connections v0.3.0 // indirect
	github.com/docker/go-metrics v0.0.0-20180209012529-399ea8c73916 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/fatih/color v1.7.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/globalsign/mgo v0.0.0-20180615134936-113d3961e731
	github.com/godbus/dbus v4.1.0+incompatible // indirect
	github.com/gogo/protobuf v1.0.0 // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2 // indirect
	github.com/gorilla/websocket v1.2.0
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/hashicorp/errwrap v0.0.0-20141028054710-7554cd9344ce // indirect
	github.com/hashicorp/go-multierror v0.0.0-20171204182908-b7773ae21874 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/j-keck/arping v0.0.0-20160618110441-2cf9dc699c56 // indirect
	github.com/kelseyhightower/envconfig v1.3.0
	github.com/kr/pty v1.1.3
	github.com/kubernetes-sigs/cri-o v0.0.0-20180917213123-8afc34092907
	github.com/magiconair/properties v1.8.0
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mattn/go-runewidth v0.0.2 // indirect
	github.com/mattn/go-shellwords v1.0.3 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mtrmac/gpgme v0.0.0-20170102180018-b2432428689c // indirect
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v0.0.0-20180411145040-e562b0440392
	github.com/opencontainers/image-tools v0.0.0-20180129025323-c95f76cbae74
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/opencontainers/runtime-spec v0.0.0-20180913141938-5806c3563733
	github.com/opencontainers/runtime-tools v0.6.0
	github.com/opencontainers/selinux v1.0.0-rc1
	github.com/pelletier/go-toml v1.2.0
	github.com/pkg/errors v0.8.0
	github.com/pquerna/ffjson v0.0.0-20171002144729-d49c2bc1aa13 // indirect
	github.com/prometheus/client_golang v0.0.0-20180607123607-faf4ec335fe0 // indirect
	github.com/prometheus/client_model v0.0.0-20171117100541-99fa1f4be8e5 // indirect
	github.com/prometheus/common v0.0.0-20180518154759-7600349dcfe1 // indirect
	github.com/prometheus/procfs v0.0.0-20180612222113-7d6f385de8be // indirect
	github.com/safchain/ethtool v0.0.0-20180504150752-6e3f4faa84e1 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/seccomp/libseccomp-golang v0.9.0
	github.com/sirupsen/logrus v1.0.5 // indirect
	github.com/spf13/cobra v0.0.0-20190321000552-67fc4837d267
	github.com/spf13/pflag v1.0.3
	github.com/sylabs/json-resp v0.5.0
	github.com/sylabs/scs-key-client v0.2.0
	github.com/sylabs/sif v1.0.3
	github.com/syndtr/gocapability v0.0.0-20180223013746-33e07d32887e // indirect
	github.com/vishvananda/netlink v1.0.0 // indirect
	github.com/vishvananda/netns v0.0.0-20171111001504-be1fbeda1936 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v0.0.0-20180618132009-1d523034197f // indirect
	go4.org v0.0.0-20180417224846-9599cf28b011 // indirect
	golang.org/x/crypto v0.0.0-20181203042331-505ab145d0a9
	golang.org/x/sync v0.0.0-20190227155943-e225da77a7e6 // indirect
	golang.org/x/sys v0.0.0-20190222072716-a9d3bda3a223
	gopkg.in/airbrake/gobrake.v2 v2.0.9 // indirect
	gopkg.in/cheggaaa/pb.v1 v1.0.25
	gopkg.in/gemnasium/logrus-airbrake-hook.v2 v2.1.2 // indirect
	gopkg.in/yaml.v2 v2.2.2
	gotest.tools v2.3.0+incompatible // indirect
	k8s.io/client-go v11.0.0+incompatible // indirect
)

replace (
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.0.5
	github.com/opencontainers/image-tools => github.com/sylabs/image-tools v0.0.0-20181006203805-2814f4980568
	golang.org/x/crypto => github.com/sylabs/golang-x-crypto v0.0.0-20181006204705-4bce89e8e9a9
)
