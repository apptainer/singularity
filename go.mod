module github.com/sylabs/singularity

go 1.13

require (
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f // indirect
	github.com/apex/log v1.1.2
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/buger/jsonparser v0.0.0-20191004114745-ee4c978eae7e
	github.com/bugsnag/bugsnag-go v1.5.1 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/containerd/cgroups v0.0.0-20200116170754-a8908713319d
	github.com/containerd/containerd v1.3.2
	github.com/containernetworking/cni v0.7.1
	github.com/containernetworking/plugins v0.8.5
	github.com/containers/image/v5 v5.2.1
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/creack/pty v1.1.9 // indirect
	github.com/deislabs/oras v0.8.1
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/fatih/color v1.9.0
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/godbus/dbus v4.1.0+incompatible // indirect
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/gorilla/handlers v1.4.0 // indirect
	github.com/gorilla/websocket v1.4.1
	github.com/juju/errors v0.0.0-20190207033735-e65537c515d7 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kr/pty v1.1.8
	github.com/mattn/go-runewidth v0.0.2 // indirect
	github.com/openSUSE/umoci v0.4.5
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/opencontainers/image-spec v1.0.2-0.20191218002246-9ea04d1f37d7
	github.com/opencontainers/image-tools v0.0.0-20180129025323-c95f76cbae74
	github.com/opencontainers/runtime-spec v1.0.2-0.20191230192225-e09c7c488e29
	github.com/opencontainers/selinux v1.3.1
	github.com/pelletier/go-toml v1.6.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v0.0.0-20180607123607-faf4ec335fe0 // indirect
	github.com/prometheus/common v0.0.0-20180518154759-7600349dcfe1 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/seccomp/containers-golang v0.3.2
	github.com/seccomp/libseccomp-golang v0.9.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/sylabs/json-resp v0.6.0
	github.com/sylabs/scs-build-client v0.1.0
	github.com/sylabs/scs-key-client v0.4.1
	github.com/sylabs/scs-library-client v0.4.4
	github.com/sylabs/sif v1.0.9
	github.com/urfave/cli v1.22.2 // indirect
	github.com/vishvananda/netlink v1.0.1-0.20190618143317-99a56c251ae6 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.6 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	go.opencensus.io v0.22.2 // indirect
	go4.org v0.0.0-20180417224846-9599cf28b011 // indirect
	golang.org/x/crypto v0.0.0-20200128174031-69ecbb4d6d5d
	golang.org/x/net v0.0.0-20200114155413-6afb5195e5aa // indirect
	golang.org/x/sys v0.0.0-20200124204421-9fbb57f87de9
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/genproto v0.0.0-20200122232147-0452cf42e150 // indirect
	gopkg.in/cheggaaa/pb.v1 v1.0.28
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools/v3 v3.0.1
	k8s.io/client-go v0.0.0-20181010045704-56e7a63b5e38 // indirect
)

replace (
	github.com/opencontainers/image-tools => github.com/sylabs/image-tools v0.0.0-20181006203805-2814f4980568
	golang.org/x/crypto => github.com/sylabs/golang-x-crypto v0.0.0-20181006204705-4bce89e8e9a9
)
