# a list of extra files to clean augmented by each module (*.mconf) file
CLEANFILES :=

# general build-wide compile options
AFLAGS := -g

LDFLAGS += -Wl,-z,relro,-z,now
