module github.com/sylabs/singularity

go 1.13

require (
	github.com/AdamKorcz/go-fuzz-headers v0.0.0-20210319161527-f761c2329661 // indirect
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667
	github.com/adigunhammedolalekan/registry-auth v0.0.0-20200730122110-8cde180a3a60
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f // indirect
	github.com/apex/log v1.9.0
	github.com/blang/semver/v4 v4.0.0
	github.com/buger/jsonparser v1.1.1
	github.com/bugsnag/bugsnag-go v1.5.1 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/containerd/cgroups v0.0.0-20200531161412-0dbf7f05ba59
	github.com/containerd/containerd v1.4.4
	github.com/containernetworking/cni v0.8.1
	github.com/containernetworking/plugins v0.9.1
	github.com/containers/image/v5 v5.10.5
	github.com/deislabs/oras v0.10.0
	github.com/fatih/color v1.10.0
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/go-log/log v0.2.0
	github.com/godbus/dbus v4.1.0+incompatible // indirect
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gorilla/handlers v1.4.0 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/klauspost/compress v1.11.13 // indirect
	github.com/kr/pty v1.1.8
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2-0.20210331164927-859973e32cca
	github.com/opencontainers/runc v1.0.0-rc93 // indirect
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/opencontainers/selinux v1.8.0
	github.com/opencontainers/umoci v0.4.7
	github.com/pelletier/go-toml v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/seccomp/containers-golang v0.6.0
	github.com/seccomp/libseccomp-golang v0.9.1
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/sylabs/json-resp v0.7.0
	github.com/sylabs/scs-build-client v0.1.5
	github.com/sylabs/scs-key-client v0.6.0
	github.com/sylabs/scs-library-client v1.0.3
	github.com/sylabs/sif v1.2.1
	github.com/urfave/cli v1.22.5 // indirect
	github.com/vbauerster/mpb/v4 v4.12.2
	github.com/vbauerster/mpb/v6 v6.0.2
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.6 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	go.opencensus.io v0.22.2 // indirect
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/sys v0.0.0-20210403161142-5e06dd20ab57
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/v3 v3.0.3
	mvdan.cc/sh/v3 v3.2.2
)

replace (
	// These are required for oras v0.9.0
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible

	golang.org/x/crypto => github.com/sylabs/golang-x-crypto v0.0.0-20181006204705-4bce89e8e9a9
)
