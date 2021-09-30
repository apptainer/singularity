module github.com/hpcng/singularity

go 1.13

require (
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667
	github.com/adigunhammedolalekan/registry-auth v0.0.0-20200730122110-8cde180a3a60
	github.com/apex/log v1.9.0
	github.com/blang/semver/v4 v4.0.0
	github.com/buger/jsonparser v1.1.1
	github.com/containerd/cgroups v1.0.1
	github.com/containerd/containerd v1.5.6
	github.com/containernetworking/cni v0.8.1
	github.com/containernetworking/plugins v0.9.1
	github.com/containers/image/v5 v5.16.0
	github.com/cyphar/filepath-securejoin v0.2.3
	github.com/fatih/color v1.13.0
	github.com/go-log/log v0.2.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.4.2
	github.com/hpcng/sif v1.6.0
	github.com/kr/pty v1.1.8
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2-0.20210819154149-5ad6f50d6283
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/opencontainers/runtime-tools v0.9.1-0.20210326182921-59cdde06764b
	github.com/opencontainers/selinux v1.8.5
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
	golang.org/x/crypto v0.0.0-20210920023735-84f357641f63
	golang.org/x/sys v0.0.0-20210820121016-41cdb8703e55
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/v3 v3.0.3
	mvdan.cc/sh/v3 v3.3.1
	oras.land/oras-go v0.4.0
)

replace (
	// These are required for oras.land/oras-go
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible

	golang.org/x/crypto => github.com/hpcng/golang-x-crypto v0.0.0-20210830200829-e6b35e3fb874
)
