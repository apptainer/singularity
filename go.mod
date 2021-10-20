module github.com/hpcng/singularity

go 1.16

require (
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667
	github.com/adigunhammedolalekan/registry-auth v0.0.0-20200730122110-8cde180a3a60
	github.com/apex/log v1.9.0
	github.com/blang/semver/v4 v4.0.0
	github.com/buger/jsonparser v1.1.1
	github.com/containerd/cgroups v1.0.2
	github.com/containerd/containerd v1.5.7
	github.com/containernetworking/cni v0.8.1
	github.com/containernetworking/plugins v0.9.1
	github.com/containers/image/v5 v5.16.1
	github.com/cyphar/filepath-securejoin v0.2.3
	github.com/docker/docker v20.10.9+incompatible
	github.com/fatih/color v1.13.0
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/go-log/log v0.2.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.4.2
	github.com/hpcng/sif v1.6.0
	github.com/kr/pty v1.1.8
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2-0.20210819154149-5ad6f50d6283
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/opencontainers/runtime-tools v0.9.1-0.20210326182921-59cdde06764b
	github.com/opencontainers/selinux v1.9.1
	github.com/opencontainers/umoci v0.4.7
	github.com/pelletier/go-toml v1.9.4
	github.com/pkg/errors v0.9.1
	github.com/satori/go.uuid v1.2.1-0.20180404165556-75cca531ea76
	github.com/seccomp/containers-golang v0.6.0
	github.com/seccomp/libseccomp-golang v0.9.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/sylabs/json-resp v0.8.0
	github.com/sylabs/scs-build-client v0.2.1
	github.com/sylabs/scs-key-client v0.6.2
	github.com/sylabs/scs-library-client v1.0.5
	github.com/vbauerster/mpb/v4 v4.12.2
	github.com/vbauerster/mpb/v6 v6.0.4
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.6 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	golang.org/x/crypto v0.0.0-20210920023735-84f357641f63
	golang.org/x/sys v0.0.0-20210925032602-92d5a993a665
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/v3 v3.0.3
	mvdan.cc/sh/v3 v3.4.0
	oras.land/oras-go v0.4.0
	rsc.io/letsencrypt v0.0.3 // indirect
)

replace golang.org/x/crypto => github.com/hpcng/golang-x-crypto v0.0.0-20210830200829-e6b35e3fb874
