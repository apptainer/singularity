CGO_CPPFLAGS += -I$(BUILDDIR) -I$(SOURCEDIR)/cmd/starter/c -I$(SOURCEDIR)/cmd/starter/c/include
CGO_CPPFLAGS += -include $(BUILDDIR_ABSPATH)/config.h

export CGO_CPPFLAGS CGO_LDFLAGS
