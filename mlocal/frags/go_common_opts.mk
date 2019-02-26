# go tool default build options
GO_TAGS := containers_image_openpgp
GO_LDFLAGS :=
GO_BUILDMODE := -buildmode=default

GOFLAGS := -mod=vendor
export GOFLAGS