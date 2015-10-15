AC_PREREQ(2.59)
AC_INIT([singularity],[0.0.1],[gmkurtzer@lbl.gov])
AC_CONFIG_SRCDIR([.])
AC_CONFIG_HEADERS([config.h])
AC_CONFIG_MACRO_DIR([.])

if test -z "$prefix" -o "$prefix" = "NONE" ; then
  prefix=${ac_default_prefix}
fi
AC_SUBST(PREFIX, $prefix)

AC_PROG_INSTALL
AM_INIT_AUTOMAKE
AC_PROG_LIBTOOL
AC_PROG_CC

dnl# Add default flags
dnlCFLAGS="-std=gnu99 $CFLAGS"

AC_CONFIG_FILES([
   Makefile
   singularity.spec
   src/Makefile
   bin/Makefile
])
AC_OUTPUT
