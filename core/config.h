#define PACKAGE_NAME "singularity"
#define PACKAGE_TARNAME "singularity"
#define PACKAGE_VERSION "3.0"
#define PACKAGE_STRING "singularity 3.0"
#define PACKAGE_BUGREPORT "gmkurtzer@gmail.com"
#define PACKAGE_URL ""

#define PREFIX "/usr/local"
#define EXECPREFIX PREFIX
#define BINDIR EXECPREFIX "/bin"
#define SBINDIR EXECPREFIX "/sbin"
#define LIBEXECDIR EXECPREFIX "/libexec"
#define DATAROOTDIR PREFIX "/share"
#define DATADIR DATAROOTDIR
#define SYSCONFDIR PREFIX "/etc"
#define SHAREDSTARTEDIR PREFIX "/com"
#define LOCALSTATEDIR PREFIX "/var"
#define INCLUDEDIR PREFIX "/include"
#define OLDINCLUDEDIR "/usr/include"
#define DOCDIR DATAROOTDIR "/doc/" PACKAGE_TARNAME
#define INFODIR DATAROOTDIR "/info"
#define HTMLDIR DOCDIR
#define DVIDIR DOCDIR
#define PDFDIR DOCDIR
#define PSDIR DOCDIR
#define LIBDIR EXECPREFIX "/lib"
#define LOCALEDIR DATAROOTDIR "/locale"
#define MANDIR DATAROOTDIR "/man"

/* check for these ! */
#define NS_CLONE_NEWPID
#define NS_CLONE_PID
#define NS_CLONE_FS
#define NS_CLONE_NEWNS
#define NS_CLONE_NEWUSER
#define NS_CLONE_NEWIPC
#define NS_CLONE_NEWNET
#define NS_CLONE_NEWUTS
#define DSINGULARITY_NO_NEW_PRIVS
#define SINGULARITY_MS_SLAVE
#define USER_CAPABILITIES
#define SINGULARITY_SECUREBITS
#define SINGULARITY_USERNS
#define SINGULARITY_NO_SETNS
#define SINGULARITY_SETNS_SYSCALL

#define CONTAINER_FINALDIR LOCALSTATEDIR "/singularity/mnt/final"
