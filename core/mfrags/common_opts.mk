# a list of extra files to clean augmented by each module (*.mconf) file
CLEANFILES :=

# general build-wide compile options
AFLAGS := -g

CFLAGS := -Wall -Werror -Wfatal-errors  -Wno-unknown-warning-option
CFLAGS += -Wstrict-prototypes -Wshadow -Wpointer-arith -Wbad-function-cast
CFLAGS += -Wwrite-strings -Woverlength-strings -Wstrict-prototypes
CFLAGS += -Wunreachable-code -Wframe-larger-than=2047 -Wno-sign-compare
CFLAGS += -Wclobbered -Wempty-body -Wimplicit-fallthrough=3
CFLAGS += -Wmissing-field-initializers -Wmissing-parameter-type -Wtype-limits
CFLAGS += -Wshift-negative-value -Wunused-parameter -Wunused-but-set-parameter
CFLAGS += -Wno-discarded-qualifiers -Wno-incompatible-pointer-types
CFLAGS += -pipe -fmessage-length=0
