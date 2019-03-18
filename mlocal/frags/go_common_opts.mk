# go tool default build options
GO_TAGS := containers_image_openpgp sylog
GO_BUILDMODE := -buildmode=default

GO_LDFLAGS :=
GO_GCFLAGS := -gcflags=all=-trimpath=`dirname $(SOURCEDIR)`
GO_ASMFLAGS := -asmflags=all=-trimpath=`dirname $(SOURCEDIR)`
