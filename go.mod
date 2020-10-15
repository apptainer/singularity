module github.com/sylabs/singularity

go 1.13

require (
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667
	github.com/adigunhammedolalekan/registry-auth v0.0.0-20200730122110-8cde180a3a60
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f // indirect
	github.com/apex/log v1.9.0
	github.com/blang/semver/v4 v4.0.0
	github.com/buger/jsonparser v1.0.0
	github.com/bugsnag/bugsnag-go v1.5.1 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/containerd/cgroups v0.0.0-20200116170754-a8908713319d
	github.com/containerd/containerd v1.4.1
	github.com/containernetworking/cni v0.8.0
	github.com/containernetworking/plugins v0.8.7
	github.com/containers/image/v5 v5.6.0
	github.com/deislabs/oras v0.8.1
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/fatih/color v1.9.0
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/go-log/log v0.2.0
	github.com/godbus/dbus v4.1.0+incompatible // indirect
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/gorilla/handlers v1.4.0 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kr/pty v1.1.8
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2-0.20191218002246-9ea04d1f37d7
	github.com/opencontainers/image-tools v0.0.0-20180129025323-c95f76cbae74
	github.com/opencontainers/runtime-spec v1.0.3-0.20200710190001-3e4195d92445
	github.com/opencontainers/selinux v1.6.0
	github.com/opencontainers/umoci v0.4.6-0.20200622135030-30d116059d97
	github.com/pelletier/go-toml v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/satori/go.uuid v1.2.0
	github.com/seccomp/containers-golang v0.6.0
	github.com/seccomp/libseccomp-golang v0.9.1
	github.com/spf13/cobra v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/sylabs/json-resp v0.7.0
	github.com/sylabs/scs-build-client v0.1.5
	github.com/sylabs/scs-key-client v0.5.1
	github.com/sylabs/scs-library-client v0.5.7
	github.com/sylabs/sif v1.2.1
	github.com/vbauerster/mpb/v4 v4.12.2
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.6 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	go.opencensus.io v0.22.2 // indirect
	go4.org v0.0.0-20180417224846-9599cf28b011 // indirect
	golang.org/x/crypto v0.0.0-20200604202706-70a84ac30bf9
	golang.org/x/sys v0.0.0-20200810151505-1b9f1253b3ed
	gopkg.in/yaml.v2 v2.3.0
	gotest.tools/v3 v3.0.3
	mvdan.cc/sh/v3 v3.1.2
	rsc.io/letsencrypt v0.0.3 // indirect
)

replace (
	github.com/opencontainers/image-tools => github.com/sylabs/image-tools v0.0.0-20181006203805-2814f4980568
	golang.org/x/crypto => github.com/sylabs/golang-x-crypto v0.0.0-20181006204705-4bce89e8e9a9
)
