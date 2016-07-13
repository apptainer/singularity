AC_PREREQ(2.59)
AC_INIT([singularity],[2.1],[gmkurtzer@lbl.gov])


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
        AC_MSG_ERROR([Architecture $target_cpu not supported])
    ;;
esac
AC_SUBST(SINGULARITY_ARCH)

AM_INIT_AUTOMAKE([foreign])
AC_CONFIG_SRCDIR([.])
AC_CONFIG_HEADERS([src/config.h])
AC_CONFIG_MACRO_DIR([.])
AC_PROG_INSTALL
AC_PROG_LIBTOOL
AC_PROG_CC
AM_INIT_AUTOMAKE
AM_PROG_CC_C_O

AC_MSG_CHECKING([for namespace: CLONE_NEWPID])
AC_COMPILE_IFELSE([AC_LANG_PROGRAM([[#define _GNU_SOURCE
                                     #include <sched.h>
                                   ]],
                                   [[unshare(CLONE_NEWPID);]])],
                      [
                          AC_MSG_RESULT([yes])
                          NAMESPACE_DEFINES="$NAMESPACE_DEFINES -DNS_CLONE_NEWPID"
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
                          NAMESPACE_DEFINES="$NAMESPACE_DEFINES -DNS_CLONE_PID"
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
                          NAMESPACE_DEFINES="$NAMESPACE_DEFINES -DNS_CLONE_FS"
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
                          NAMESPACE_DEFINES="$NAMESPACE_DEFINES -DNS_CLONE_NEWNS"
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


AC_SUBST(NAMESPACE_DEFINES)

AC_CHECK_FUNCS(setns, [
                      ], [
                          NO_SETNS="-DNO_SETNS"
                      ]
                  )


AC_SUBST(NO_SETNS)

#AC_CHECK_DECLS([MS_PRIVATE,MS_REC], [],
#               [AC_MSG_ERROR([Required mount(2) flags not available])],
#               [[#include <sys/mount.h>]])


AC_CONFIG_FILES([
   Makefile
   singularity.spec
   src/Makefile
   etc/Makefile
   bin/Makefile
   bin/singularity
   libexec/Makefile
   libexec/mods/Makefile
   libexec/cli/Makefile
])
AC_OUTPUT
