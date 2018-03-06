# a list of extra files to clean augmented by each module (*.mconf) file
CLEANFILES :=

# general build-wide compile options
AFLAGS := -g

CFLAGS := -ansi -Wall -Wstrict-prototypes -Wextra -Wunused -Werror
CGLAGS += -Wuninitialized -Wshadow -Wpointer-arith -Wbad-function-cast
CFLAGS += -Wwrite-strings -Woverlength-strings -Wstrict-prototypes
CFLAGS += -Wunreachable-code -Wframe-larger-than=2047 -Wno-discarded-qualifiers
CFLAGS += -Wfatal-errors -pipe -fmessage-length=0 -fno-builtin -nostdinc
CFLAGS += -fplan9-extensions

CPPFLAGS += -I$(SOURCEDIR)/$(ARCH)/include
CPPFLAGS += -I$(SOURCEDIR)/port
QEMUOPTS := -machine q35 -enable-kvm -cpu host -smp cpus=2 -m 1024 \
	-device ioh3420,id=root.0,slot=1 \
	-device x3130-upstream,bus=root.0,id=upstream1 \
	-device xio3130-downstream,bus=upstream1,id=downstream1,chassis=1 \
	-device ioh3420,id=root.1,slot=2 \
	-device virtio-scsi-pci,id=virtio-scsi0,disable-legacy=on,disable-modern=off \
	-drive if=none,cache=none,file=$(BUILDDIR)/pc/bootdisk,format=raw,id=drive0 \
	-device scsi-hd,drive=drive0 \
	-device virtio-balloon-pci,id=balloon0,bus=root.0 \
	-device virtio-keyboard-pci,id=keyboard0,bus=root.0 \
	-device virtio-mouse-pci,id=mouse0,bus=root.0 \
	-netdev type=user,id=net0 \
	-device virtio-net-pci,netdev=net0,bus=root.0 \
	-device virtio-vga,id=gpu0,bus=root.0
QEMUDBGOPTS := -m 16 -drive file=$(BUILDDIR)/pc/bootdisk,if=virtio,format=raw

GDB=cgdb
