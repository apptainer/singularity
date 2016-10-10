AC_PREREQ(2.59)
AC_INIT([singularity],[2.2],[gmkurtzer@lbl.gov])

if test -z "$prefix" -o "$prefix" = "NONE" ; then
  prefix=${ac_default_prefix}
fi
AC_SUBST(PREFIX, $prefix)

AC_CANONICAL_TARGET
case $target_cpu in
    x86_64)
        SINGULARITY_ARCH=x86_64
    ;;
    i?86)
        SINGULARITY_ARCH=i386
    ;;
    athlon)
        SINGULARITY_ARCH=i386
    ;;
    *)
        SINGULARITY_ARCH="$target_cpu"
    ;;
esac
AC_SUBST(SINGULARITY_ARCH)

# This should be used per autogen.sh output, but we hit this automake bug:
# https://www.mail-archive.com/debian-bugs-dist@lists.debian.org/msg1232579.html
#AM_INIT_AUTOMAKE([foreign subdir-objects])

AM_INIT_AUTOMAKE([foreign subdir-objects]) ## SEE ABOVE BEFORE CHANGING
AC_CONFIG_SRCDIR([src])
AC_CONFIG_HEADERS([src/config.h])
AC_CONFIG_MACRO_DIR([m4])
AC_PROG_INSTALL
AC_PROG_LIBTOOL
AC_PROG_CC
AM_INIT_AUTOMAKE
AM_PROG_CC_C_O
AC_ENABLE_SHARED
AC_PROG_LIBTOOL(libtool)

# Setting rpath if necessary
if test "$libdir" = "\${exec_prefix}/lib"; then
    LDFLAGS="$LDFLAGS -Wl,-rpath -Wl,\$(libdir)"
fi

AC_MSG_CHECKING([for namespace: CLONE_NEWPID])
AC_COMPILE_IFELSE([AC_LANG_PROGRAM([[#define _GNU_SOURCE
                                     #include <sched.h>
                                   ]],
                                   [[unshare(CLONE_NEWPID);]])],
                      [
                          AC_MSG_RESULT([yes])
                          SINGULARITY_DEFINES="$SINGULARITY_DEFINES -DNS_CLONE_NEWPID"
                      ], [
                          AC_MSG_RESULT([no])
                      ]
                  )


AC_MSG_CHECKING([for namespace: CLONE_PID])
AC_COMPILE_IFELSE([AC_LANG_PROGRAM([[#define _GNU_SOURCE
                                     #include <sched.h>
                                   ]],
                                   [[unshare(CLONE_PID);]])],
                      [
                          AC_MSG_RESULT([yes])
                          SINGULARITY_DEFINES="$SINGULARITY_DEFINES -DNS_CLONE_PID"
                      ], [
                          AC_MSG_RESULT([no])
                      ]
                  )


AC_MSG_CHECKING([for namespace: CLONE_FS])
AC_COMPILE_IFELSE([AC_LANG_PROGRAM([[#define _GNU_SOURCE
                                     #include <sched.h>
                                   ]],
                                   [[unshare(CLONE_FS);]])],
                      [
                          AC_MSG_RESULT([yes])
                          SINGULARITY_DEFINES="$SINGULARITY_DEFINES -DNS_CLONE_FS"
                      ], [
                          AC_MSG_RESULT([no])
                      ]
                  )


AC_MSG_CHECKING([for namespace: CLONE_NEWNS])
AC_COMPILE_IFELSE([AC_LANG_PROGRAM([[#define _GNU_SOURCE
                                     #include <sched.h>
                                   ]],
                                   [[unshare(CLONE_NEWNS);]])],
                      [
                          AC_MSG_RESULT([yes])
                          SINGULARITY_DEFINES="$SINGULARITY_DEFINES -DNS_CLONE_NEWNS"
                      ], [
                          AC_MSG_RESULT([no])
                          echo
                          echo "ERROR!!!!!!"
                          echo
                          echo "This host does not support the CLONE_NEWNS (mount) namespace flag! You"
                          echo "really really really don't want to run Singularity containers without a"
                          echo "Separate mount name namespace!"
                          echo
                          exit 255
                      ]
                  )


AC_MSG_CHECKING([for namespace: CLONE_NEWUSER])
AC_COMPILE_IFELSE([AC_LANG_PROGRAM([[#define _GNU_SOURCE
                                     #include <sched.h>
                                   ]],
                                   [[unshare(CLONE_NEWUSER);]])],
                      [
                          AC_MSG_RESULT([yes])
                          SINGULARITY_DEFINES="$SINGULARITY_DEFINES -DNS_CLONE_NEWUSER"
                      ], [
                          AC_MSG_RESULT([no])
                      ]
                  )


AC_MSG_CHECKING([for namespace: CLONE_NEWIPC])
AC_COMPILE_IFELSE([AC_LANG_PROGRAM([[#define _GNU_SOURCE
                                     #include <sched.h>
                                   ]],
                                   [[unshare(CLONE_NEWIPC);]])],
                      [
                          AC_MSG_RESULT([yes])
                          SINGULARITY_DEFINES="$SINGULARITY_DEFINES -DNS_CLONE_NEWIPC"
                      ], [
                          AC_MSG_RESULT([no])
                      ]
                  )


AC_MSG_CHECKING([for feature: NO_NEW_PRIVS])
AC_COMPILE_IFELSE([AC_LANG_PROGRAM([[
                                     #include <sys/prctl.h>
                                   ]],
                                   [[prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0);
                                     prctl(PR_GET_NO_NEW_PRIVS, 0, 0, 0, 0);]])],
                      [
                          AC_MSG_RESULT([yes])
                          SINGULARITY_DEFINES="$SINGULARITY_DEFINES -DSINGULARITY_NO_NEW_PRIVS"
                      ], [
                          AC_MSG_RESULT([no])
                      ]
                  )


AC_MSG_CHECKING([for feature: MS_SLAVE])
AC_COMPILE_IFELSE([AC_LANG_PROGRAM([[
                                     #include <sys/mount.h>
                                   ]],
                                   [[#ifndef MS_SLAVE
                                     #error failed
                                     #endif
                                   ]])],
                      [
                          AC_MSG_RESULT([yes])
                          SINGULARITY_DEFINES="$SINGULARITY_DEFINES -DSINGULARITY_MS_SLAVE"
                      ], [
                          AC_MSG_RESULT([no])
                      ]
                  )

AC_MSG_CHECKING([for feature: MS_REC])
AC_COMPILE_IFELSE([AC_LANG_PROGRAM([[
                                     #include <sys/mount.h>
                                   ]],
                                   [[#ifndef MS_REC
                                     #error failed
                                     #endif
                                   ]])],
                      [
                          AC_MSG_RESULT([yes])
                      ], [
                          AC_MSG_RESULT([no])
                          echo
                          echo "ERROR!!!!!!"
                          echo
                          echo "This host does not support the MS_REC mount option!"
                          echo
                          exit 255
                      ]
                  )


AC_MSG_CHECKING([for feature: MS_PRIVATE])
AC_COMPILE_IFELSE([AC_LANG_PROGRAM([[
                                     #include <sys/mount.h>
                                   ]],
                                   [[#ifndef MS_PRIVATE
                                     #error failed
                                     #endif
                                   ]])],
                      [
                          AC_MSG_RESULT([yes])
                      ], [
                          AC_MSG_RESULT([no])
                          echo
                          echo "ERROR!!!!!!"
                          echo
                          echo "This host does not support the MS_PRIVATE mount option!"
                          echo
                          exit 255
                      ]
                  )



AC_ARG_WITH([userns],
            AS_HELP_STRING([--with-userns], [Enable use of user namespaces]),
            [SINGULARITY_DEFINES="$SINGULARITY_DEFINES -DSINGULARITY_USERNS"]
           )


AC_CHECK_FUNCS(setns, [
                      ], [
                          NO_SETNS="-DNO_SETNS"
                      ]
                  )


AC_SUBST(NO_SETNS)
AC_MSG_CHECKING([for overlayfs])
KVERS=`uname -r`
if test -f "/lib/modules/$KVERS/modules.dep"; then
    if grep -q 'overlay.ko' "/lib/modules/$KVERS/modules.dep"; then
        AC_MSG_RESULT([yes])
        SINGULARITY_DEFINES="$SINGULARITY_DEFINES -DSINGULARITY_OVERLAYFS"
        OVERLAY_FS=1
    else
        AC_MSG_RESULT([no])
    fi
else
    AC_MSG_RESULT([maybe])
    SINGULARITY_DEFINES="$SINGULARITY_DEFINES -DSINGULARITY_OVERLAYFS"
fi

AC_SUBST(OVERLAY_FS)
AC_SUBST(SINGULARITY_DEFINES)



AC_MSG_CHECKING([if suid should be enabled])
AC_ARG_ENABLE([suid],
    AS_HELP_STRING([--disable-suid], [Disable build for SUID (only works for new kernels)]))

AS_IF([test "x$enable_suid" != "xno"], [
    AC_MSG_RESULT([yes])
    BUILD_SUID="sexec-suid"
], [
    AC_MSG_RESULT([no])
    BUILD_SUID=""
])

AC_SUBST(BUILD_SUID)


#AC_CHECK_DECLS([MS_PRIVATE,MS_REC], [],
#               [AC_MSG_ERROR([Required mount(2) flags not available])],
#               [[#include <sys/mount.h>]])




AC_CONFIG_FILES([
   Makefile
   singularity.spec
   test.sh
   src/Makefile
   src/lib/Makefile
   src/lib/ns/Makefile
   src/lib/ns/ipc/Makefile
   src/lib/ns/mnt/Makefile
   src/lib/ns/pid/Makefile
   src/lib/ns/user/Makefile
   src/lib/rootfs/Makefile
   src/lib/rootfs/image/Makefile
   src/lib/rootfs/dir/Makefile
   src/lib/rootfs/squashfs/Makefile
   src/lib/action/Makefile
   src/lib/action/exec/Makefile
   src/lib/action/shell/Makefile
   src/lib/action/start/Makefile
   src/lib/action/stop/Makefile
   src/lib/action/test/Makefile
   src/lib/action/run/Makefile
   src/lib/file/Makefile
   src/lib/file/group/Makefile
   src/lib/file/passwd/Makefile
   src/lib/file/resolvconf/Makefile
   src/lib/mount/Makefile
   src/lib/mount/cwd/Makefile
   src/lib/mount/dev/Makefile
   src/lib/mount/binds/Makefile
   src/lib/mount/home/Makefile
   src/lib/mount/hostfs/Makefile
   src/lib/mount/kernelfs/Makefile
   src/lib/mount/tmp/Makefile
   src/lib/mount/userbinds/Makefile
   src/lib/mount/scratch/Makefile
   src/util/Makefile
   etc/Makefile
   bin/Makefile
   bin/singularity
   man/Makefile
   libexec/Makefile
   libexec/bootstrap/Makefile
   libexec/bootstrap/modules-v1/Makefile
   libexec/bootstrap/modules-v2/Makefile
   libexec/cli/Makefile
   libexec/helpers/Makefile
   libexec/python/Makefile
   libexec/python/docker/Makefile
])
AC_OUTPUT
