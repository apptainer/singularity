# umoci: Umoci Modifies Open Containers' Images
# Copyright (C) 2016, 2017, 2018 SUSE LLC.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM opensuse/amd64:42.3
MAINTAINER "Aleksa Sarai <asarai@suse.com>"

# We have to use out-of-tree repos because several packages haven't been merged
# into openSUSE:Factory yet.
RUN zypper ar -f -p 10 -g obs://Virtualization:containers obs-vc && \
	zypper --gpg-auto-import-keys -n ref && \
	zypper -n up
RUN zypper -n in \
		bats \
		git \
		go \
		golang-github-cpuguy83-go-md2man \
		go-mtree \
		jq \
		libcap-progs \
		make \
		moreutils \
		oci-image-tools \
		oci-runtime-tools \
		python-setuptools python-xattr attr \
		skopeo \
		tar

ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH
RUN go get -u github.com/golang/lint/golint
RUN go get -u github.com/vbatts/git-validation && type git-validation

ENV SOURCE_IMAGE=/opensuse SOURCE_TAG=latest
ARG DOCKER_IMAGE=opensuse/amd64:tumbleweed
RUN skopeo copy docker://$DOCKER_IMAGE oci:$SOURCE_IMAGE:$SOURCE_TAG

VOLUME ["/go/src/github.com/openSUSE/umoci"]
WORKDIR /go/src/github.com/openSUSE/umoci
COPY . /go/src/github.com/openSUSE/umoci
