module github.com/sylabs/singularity

go 1.11

require (
	github.com/Netflix/go-expect v0.0.0-20180928190340-9d1f4485533b
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f // indirect
	github.com/apex/log v1.1.1 // indirect
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973 // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bshuster-repo/logrus-logstash-hook v0.4.1 // indirect
	github.com/buger/jsonparser v0.0.0-20181115193947-bf1c66bbce23
	github.com/bugsnag/bugsnag-go v1.5.1 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/containerd/cgroups v0.0.0-20181208203134-65ce98b3dfeb
	github.com/containerd/containerd v1.2.9
	github.com/containerd/continuity v0.0.0-20180612233548-246e49050efd // indirect
	github.com/containernetworking/cni v0.7.1
	github.com/containernetworking/plugins v0.8.2
	github.com/containers/image v0.0.0-20180612162315-2e4f799f5eba
	github.com/containers/storage v0.0.0-20180604200230-88d80428f9b1 // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/deislabs/oras v0.4.0
	github.com/docker/distribution v0.0.0-20180611183926-749f6afb4572 // indirect
	github.com/docker/docker v0.0.0-20180522102801-da99009bbb11
	github.com/docker/docker-credential-helpers v0.6.0 // indirect
	github.com/docker/go-connections v0.3.0 // indirect
	github.com/docker/go-metrics v0.0.0-20180209012529-399ea8c73916 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/fatih/color v1.7.0
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/godbus/dbus v4.1.0+incompatible // indirect
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/handlers v1.4.0 // indirect
	github.com/gorilla/mux v1.6.2 // indirect
	github.com/gorilla/websocket v1.4.1
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/hashicorp/errwrap v0.0.0-20141028054710-7554cd9344ce // indirect
	github.com/hashicorp/go-multierror v0.0.0-20171204182908-b7773ae21874 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/juju/errors v0.0.0-20190207033735-e65537c515d7 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/klauspost/pgzip v1.2.1 // indirect
	github.com/kr/pty v1.1.8
	github.com/kubernetes-sigs/cri-o v0.0.0-20180917213123-8afc34092907
	github.com/mattn/go-runewidth v0.0.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/mtrmac/gpgme v0.0.0-20170102180018-b2432428689c // indirect
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/openSUSE/umoci v0.4.2
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/opencontainers/image-spec v0.0.0-20180411145040-e562b0440392
	github.com/opencontainers/image-tools v0.0.0-20180129025323-c95f76cbae74
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/opencontainers/runtime-spec v0.1.2-0.20181111125026-1722abf79c2f
	github.com/opencontainers/runtime-tools v0.9.0
	github.com/opencontainers/selinux v1.3.0
	github.com/pelletier/go-toml v1.4.0
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2 // indirect
	github.com/pkg/errors v0.8.1
	github.com/pquerna/ffjson v0.0.0-20171002144729-d49c2bc1aa13 // indirect
	github.com/prometheus/client_golang v0.0.0-20180607123607-faf4ec335fe0 // indirect
	github.com/prometheus/client_model v0.0.0-20171117100541-99fa1f4be8e5 // indirect
	github.com/prometheus/common v0.0.0-20180518154759-7600349dcfe1 // indirect
	github.com/prometheus/procfs v0.0.0-20180612222113-7d6f385de8be // indirect
	github.com/rootless-containers/proto v0.1.0 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/seccomp/libseccomp-golang v0.9.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stevvooe/resumable v0.0.0-20180830230917-22b14a53ba50 // indirect
	github.com/stretchr/testify v1.4.0 // indirect
	github.com/sylabs/json-resp v0.6.0
	github.com/sylabs/scs-build-client v0.0.4
	github.com/sylabs/scs-key-client v0.3.1-0.20190509220229-bce3b050c4ec
	github.com/sylabs/scs-library-client v0.4.4
	github.com/sylabs/sif v1.0.8
	github.com/syndtr/gocapability v0.0.0-20180223013746-33e07d32887e // indirect
	github.com/urfave/cli v1.21.0 // indirect
	github.com/vbatts/go-mtree v0.4.4 // indirect
	github.com/vishvananda/netlink v1.0.1-0.20190618143317-99a56c251ae6 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v0.0.0-20180618132009-1d523034197f // indirect
	github.com/xenolf/lego v2.5.0+incompatible // indirect
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.6 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	go4.org v0.0.0-20180417224846-9599cf28b011 // indirect
	golang.org/x/crypto v0.0.0-20190426145343-a29dc8fdc734
	golang.org/x/sys v0.0.0-20190616124812-15dcb6c0061f
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/grpc v1.20.0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/cheggaaa/pb.v1 v1.0.28
	gopkg.in/yaml.v2 v2.2.2
	gotest.tools v2.2.0+incompatible // indirect
	gotest.tools/v3 v3.0.0
	k8s.io/client-go v0.0.0-20181010045704-56e7a63b5e38 // indirect
	rsc.io/letsencrypt v0.0.1 // indirect
)

replace (
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.0.5
	github.com/opencontainers/image-tools => github.com/sylabs/image-tools v0.0.0-20181006203805-2814f4980568
	golang.org/x/crypto => github.com/sylabs/golang-x-crypto v0.0.0-20181006204705-4bce89e8e9a9
)
