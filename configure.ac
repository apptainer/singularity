AC_PREREQ(2.59)
AC_INIT([singularity],[2.3.1],[gmkurtzer@lbl.gov])

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
AC_GNU_SOURCE
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
            [
                SINGULARITY_DEFINES="$SINGULARITY_DEFINES -DSINGULARITY_USERNS"
                USER_NS=1
            ]
           )
AC_SUBST(USER_NS)


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


# ---------------------------------------------------------------------
# PYTHON
# ---------------------------------------------------------------------

AC_CHECK_PROG(PYTHON_CHECK,python,yes)
if test x"$PYTHON_CHECK" != x"yes" ; then
    AC_MSG_ERROR([Please install python before installing.])
else

    PYTHON_MODULES="base64 datetime glob hashlib io itertools json math multiprocessing pickle pwd re shutil signal subprocess stat sys tarfile tempfile time"
    for PYTHON_MODULE in $PYTHON_MODULES; do
        AC_MSG_CHECKING([for the $PYTHON_MODULE python module])
	    python_module_result=`python -c "import $PYTHON_MODULE" 2>&1`
	    if test -z "$python_module_result"; then
		AC_MSG_RESULT([yes])
	    else
		AC_MSG_RESULT([no])
		AC_MSG_ERROR([cannot import Python module $PYTHON_MODULE.
                              Please check your Python installation. The error was:
                              $python_module_result])
	    fi
    done

fi

AC_SUBST(OVERLAY_FS)
AC_SUBST(SINGULARITY_DEFINES)

AC_MSG_CHECKING([--with-slurm])
AC_ARG_WITH([slurm],
            AS_HELP_STRING([--with-slurm], [Enable build of slurm integration]),
           )
AS_IF([test "x$with_slurm" == "xyes"],
    [
        AC_MSG_RESULT([yes])
        AC_CHECK_HEADER([slurm/spank.h],
            [AC_MSG_RESULT([yes])
                with_slurm="yes"
            ],
            [
                AC_MSG_ERROR([SLURM support requested, but slurm/spank.h header not found.])
            ])
    ],
    [
        AC_MSG_RESULT([no])
        with_slurm="no"
    ]
)
AM_CONDITIONAL([WITH_SLURM], [test "$with_slurm" = "yes"])
AC_SUBST(with_slurm)


AC_MSG_CHECKING([if suid should be enabled])
AC_ARG_ENABLE([suid],
    AS_HELP_STRING([--disable-suid], [Disable build for SUID (only works for new kernels)]))

AS_IF([test "x$enable_suid" != "xno"], [
    AC_MSG_RESULT([yes])
    BUILD_SUID="action-suid create-suid expand-suid export-suid import-suid mount-suid"
], [
    AC_MSG_RESULT([no])
    BUILD_SUID=""
])

AC_SUBST(BUILD_SUID)


AC_MSG_CHECKING([for git version])
if BRANCH=`git rev-parse --abbrev-ref HEAD 2>/dev/null`; then
    if HASH=`git rev-parse --short HEAD 2>/dev/null`; then
        GIT_VERSION="-$BRANCH.g$HASH"
    else
        GIT_VERSION="-$BRANCH"
    fi
else
    GIT_VERSION="-dist"
fi
AC_MSG_RESULT([$GIT_VERSION])
AC_SUBST(GIT_VERSION)


AC_CONFIG_FILES([
   Makefile
   singularity.spec
   test.sh
   src/slurm/Makefile
   src/Makefile
   src/lib/Makefile
   src/lib/runtime/Makefile
   src/lib/runtime/ns/Makefile
   src/lib/runtime/ns/ipc/Makefile
   src/lib/runtime/ns/mnt/Makefile
   src/lib/runtime/ns/pid/Makefile
   src/lib/runtime/files/Makefile
   src/lib/runtime/files/group/Makefile
   src/lib/runtime/files/passwd/Makefile
   src/lib/runtime/files/resolvconf/Makefile
   src/lib/runtime/files/libs/Makefile
   src/lib/runtime/mounts/Makefile
   src/lib/runtime/mounts/cwd/Makefile
   src/lib/runtime/mounts/dev/Makefile
   src/lib/runtime/mounts/binds/Makefile
   src/lib/runtime/mounts/home/Makefile
   src/lib/runtime/mounts/hostfs/Makefile
   src/lib/runtime/mounts/kernelfs/Makefile
   src/lib/runtime/mounts/tmp/Makefile
   src/lib/runtime/mounts/userbinds/Makefile
   src/lib/runtime/mounts/scratch/Makefile
   src/lib/runtime/enter/Makefile
   src/lib/runtime/enter/chroot/Makefile
   src/lib/runtime/environment/Makefile
   src/lib/runtime/overlayfs/Makefile
   src/lib/image/Makefile
   src/lib/image/bind/Makefile
   src/lib/image/check/Makefile
   src/lib/image/check/image/Makefile
   src/lib/image/check/squashfs/Makefile
   src/lib/image/check/dir/Makefile
   src/lib/image/create/Makefile
   src/lib/image/expand/Makefile
   src/lib/image/mount/Makefile
   src/lib/image/mount/dir/Makefile
   src/lib/image/mount/image/Makefile
   src/lib/image/mount/squashfs/Makefile
   src/lib/image/offset/Makefile
   src/lib/image/open/Makefile
   src/action-lib/Makefile
   src/bootstrap-lib/Makefile
   src/util/Makefile
   etc/Makefile
   bin/Makefile
   bin/singularity
   man/Makefile
   libexec/Makefile
   libexec/bootstrap-scripts/Makefile
   libexec/bootstrap-scripts/environment/Makefile
   libexec/cli/Makefile
   libexec/helpers/Makefile
   libexec/helpers/checks/Makefile
   libexec/python/Makefile
   libexec/python/docker/Makefile
   libexec/python/shub/Makefile
   libexec/python/helpers/Makefile
   libexec/python/helpers/json/Makefile

])
AC_OUTPUT
