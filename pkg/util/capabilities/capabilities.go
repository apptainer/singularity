// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package capabilities

import "strings"

const (
	// Permitted capability string constant.
	Permitted string = "permitted"
	// Effective capability string constant.
	Effective = "effective"
	// Inheritable capability string constant.
	Inheritable = "inheritable"
	// Ambient capability string constant.
	Ambient = "ambient"
	// Bounding capability string constant.
	Bounding = "bounding"
)

type capability struct {
	Name        string
	Value       uint
	Description string
}

var (
	capChown = &capability{
		Name:  "CAP_CHOWN",
		Value: 0,
		Description: `CAP_CHOWN
	Make arbitrary changes to file UIDs and GIDs (see chown(2)).`,
	}

	capDacOverride = &capability{
		Name:  "CAP_DAC_OVERRIDE",
		Value: 1,
		Description: `CAP_DAC_OVERRIDE
	Bypass file read, write, and execute permission checks. (DAC is an abbreviation of "discretionary access control".)`,
	}

	capDacReadSearch = &capability{
		Name:  "CAP_DAC_READ_SEARCH",
		Value: 2,
		Description: `CAP_DAC_READ_SEARCH
	* Bypass file read permission checks and directory read and execute permission checks.
	* Invoke open_by_handle_at(2).`,
	}

	capFowner = &capability{
		Name:  "CAP_FOWNER",
		Value: 3,
		Description: `CAP_FOWNER
	* Bypass permission checks on operations that normally require the filesystem UID of the process to match the UID of
	  the file (e.g., chmod(2), utime(2)), excluding those operations covered by CAP_DAC_OVERRIDE and CAP_DAC_READ_SEARCH.
	* set extended file attributes (see chattr(1)) on arbitrary files.
	* set Access Control Lists (ACLs) on arbitrary files.
	* ignore directory sticky bit on file deletion.
	* specify O_NOATIME for arbitrary files in open(2) and fcntl(2).`,
	}

	capFsetid = &capability{
		Name:  "CAP_FSETID",
		Value: 4,
		Description: `CAP_FSETID
	Don't  clear set-user-ID and set-group-ID mode bits when a file is modified; set the set-group-ID bit for a file whose
	GID does not match the filesystem or any of the supplementary GIDs of the calling process.`,
	}

	capKill = &capability{
		Name:  "CAP_KILL",
		Value: 5,
		Description: `CAP_KILL
	Bypass permission checks for sending signals (see kill(2)). This includes use of the ioctl(2) KDSIGACCEPT operation.`,
	}

	capSetgid = &capability{
		Name:  "CAP_SETGID",
		Value: 6,
		Description: `CAP_SETGID
	Make arbitrary manipulations of process GIDs and supplementary GID list; forge GID when passing socket credentials via
	UNIX domain sockets; write a group ID mapping in a user namespace (see user_namespaces(7)).`,
	}

	capSetuid = &capability{
		Name:  "CAP_SETUID",
		Value: 7,
		Description: `CAP_SETUID
	Make arbitrary manipulations of process UIDs (setuid(2), setreuid(2), setresuid(2), setfsuid(2)); forge UID when pass‐
	ing socket credentials via UNIX domain sockets; write a user ID mapping in a user namespace (see user_namespaces(7)).`,
	}

	capSetpcap = &capability{
		Name:  "CAP_SETPCAP",
		Value: 8,
		Description: `CAP_SETPCAP
	If file capabilities are not supported: grant or remove any capability in the caller's permitted capability set to or
	from any other process. (This property of CAP_SETPCAP is not available when the kernel is configured to support file
	capabilities, since CAP_SETPCAP has entirely different semantics for such kernels.)

	If file capabilities are supported: add any capability from the calling thread's bounding set to its inheritable set;
	drop capabilities from the bounding set (via prctl(2) PR_CAPBSET_DROP); make changes to the securebits flags.`,
	}

	capLinuxImmutable = &capability{
		Name:  "CAP_LINUX_IMMUTABLE",
		Value: 9,
		Description: `CAP_LINUX_IMMUTABLE
	Set the FS_APPEND_FL and FS_IMMUTABLE_FL inode flags (see chattr(1)).`,
	}

	capNetBindService = &capability{
		Name:  "CAP_NET_BIND_SERVICE",
		Value: 10,
		Description: `CAP_NET_BIND_SERVICE
	Bind a socket to Internet domain privileged ports (port numbers less than 1024).`,
	}

	capNetBroadcast = &capability{
		Name:  "CAP_NET_BROADCAST",
		Value: 11,
		Description: `CAP_NET_BROADCAST
	(Unused)  Make socket broadcasts, and listen to multicasts.`,
	}

	capNetAdmin = &capability{
		Name:  "CAP_NET_ADMIN",
		Value: 12,
		Description: `CAP_NET_ADMIN
	Perform various network-related operations:
	* interface configuration.
	* administration of IP firewall, masquerading, and accounting.
	* modify routing tables.
	* bind to any address for transparent proxying.
	* set type-of-service (TOS)
	* clear driver statistics.
	* set promiscuous mode.
	* enabling multicasting.
	* use setsockopt(2) to set the following socket options: SO_DEBUG, SO_MARK, SO_PRIORITY (for a priority outside the
	  range 0 to 6), SO_RCVBUFFORCE, and SO_SNDBUFFORCE.`,
	}

	capNetRaw = &capability{
		Name:  "CAP_NET_RAW",
		Value: 13,
		Description: `CAP_NET_RAW
	* use RAW and PACKET sockets.
	* bind to any address for transparent proxying.`,
	}

	capIpcLock = &capability{
		Name:  "CAP_IPC_LOCK",
		Value: 14,
		Description: `CAP_IPC_LOCK
	Lock memory (mlock(2), mlockall(2), mmap(2), shmctl(2)).`,
	}

	capIpcOwner = &capability{
		Name:  "CAP_IPC_OWNER",
		Value: 15,
		Description: `CAP_IPC_OWNER
	Bypass permission checks for operations on System V IPC objects.`,
	}

	capSysModule = &capability{
		Name:  "CAP_SYS_MODULE",
		Value: 16,
		Description: `CAP_SYS_MODULE
	Load and unload kernel modules (see init_module(2) and delete_module(2)); in kernels before 2.6.25: drop capabilities
	from the system-wide capability bounding set.`,
	}

	capSysRawio = &capability{
		Name:  "CAP_SYS_RAWIO",
		Value: 17,
		Description: `CAP_SYS_RAWIO
	* Perform I/O port operations (iopl(2) and ioperm(2)).
	* access /proc/kcore.
	* employ the FIBMAP ioctl(2) operation.
	* open devices for accessing x86 model-specific registers (MSRs, see msr(4)).
	* update /proc/sys/vm/mmap_min_addr.
	* create memory mappings at addresses below the value specified by /proc/sys/vm/mmap_min_addr.
	* map files in /proc/bus/pci.
	* open /dev/mem and /dev/kmem.
	* perform various SCSI device commands.
	* perform certain operations on hpsa(4) and cciss(4) devices.
	* perform a range of device-specific operations on other devices.`,
	}

	capSysChroot = &capability{
		Name:  "CAP_SYS_CHROOT",
		Value: 18,
		Description: `CAP_SYS_CHROOT
	Use chroot(2).`,
	}

	capSysPtrace = &capability{
		Name:  "CAP_SYS_PTRACE",
		Value: 19,
		Description: `CAP_SYS_PTRACE
	*  Trace arbitrary processes using ptrace(2).
	*  apply get_robust_list(2) to arbitrary processes.
	*  transfer data to or from the memory of arbitrary processes using process_vm_readv(2) and process_vm_writev(2).
	*  inspect processes using kcmp(2).`,
	}

	capSysPacct = &capability{
		Name:  "CAP_SYS_PACCT",
		Value: 20,
		Description: `CAP_SYS_PACCT
	Use acct(2).`,
	}

	capSysAdmin = &capability{
		Name:  "CAP_SYS_ADMIN",
		Value: 21,
		Description: `CAP_SYS_ADMIN
	* Perform a range of system administration operations including: quotactl(2), mount(2), umount(2), swapon(2),
	  swapoff(2), sethostname(2), and setdomainname(2).
	* perform privileged syslog(2) operations (since Linux 2.6.37, CAP_SYSLOG should be used to permit such operations).
	* perform VM86_REQUEST_IRQ vm86(2) command.
	* perform IPC_SET and IPC_RMID operations on arbitrary System V IPC objects.
	* override RLIMIT_NPROC resource limit.
	* perform operations on trusted and security Extended Attributes (see xattr(7)).
	* use lookup_dcookie(2).
	* use ioprio_set(2) to assign IOPRIO_CLASS_RT and (before Linux 2.6.25) IOPRIO_CLASS_IDLE I/O scheduling classes.
	* forge PID when passing socket credentials via UNIX domain sockets.
	* exceed /proc/sys/fs/file-max, the system-wide limit on the number of open files, in system calls that open files
	  (e.g., accept(2), execve(2), open(2), pipe(2)).
	* employ CLONE_* flags that create new namespaces with clone(2) and unshare(2) (but, since Linux 3.8, creating user
	  namespaces does not require any capability).
	* call perf_event_open(2).
	* access privileged perf event information.
	* call setns(2) (requires CAP_SYS_ADMIN in the target namespace).
	* call fanotify_init(2).
	* call bpf(2).
	* perform KEYCTL_CHOWN and KEYCTL_SETPERM keyctl(2) operations.
	* perform madvise(2) MADV_HWPOISON operation.
	* employ the TIOCSTI ioctl(2) to insert characters into the input queue of a terminal other than the caller's control‐
	  ling terminal.
	* employ the obsolete nfsservctl(2) system call.
	* employ the obsolete bdflush(2) system call.
	* perform various privileged block-device ioctl(2) operations.
	* perform various privileged filesystem ioctl(2) operations.
	* perform administrative operations on many device drivers.`,
	}

	capSysBoot = &capability{
		Name:  "CAP_SYS_BOOT",
		Value: 22,
		Description: `CAP_SYS_BOOT
	Use reboot(2) and kexec_load(2).`,
	}

	capSysNice = &capability{
		Name:  "CAP_SYS_NICE",
		Value: 23,
		Description: `CAP_SYS_NICE
	* Raise process nice value (nice(2), setpriority(2)) and change the nice value for arbitrary processes.
	* set real-time scheduling policies for calling process, and set scheduling policies and priorities for arbitrary
	  processes (sched_setscheduler(2), sched_setparam(2), shed_setattr(2)).
	* set CPU affinity for arbitrary processes (sched_setaffinity(2)).
	* set I/O scheduling class and priority for arbitrary processes (ioprio_set(2)).
	* apply migrate_pages(2) to arbitrary processes and allow processes to be migrated to arbitrary nodes.
	* apply move_pages(2) to arbitrary processes.
	* use the MPOL_MF_MOVE_ALL flag with mbind(2) and move_pages(2).`,
	}

	capSysResource = &capability{
		Name:  "CAP_SYS_RESOURCE",
		Value: 24,
		Description: `CAP_SYS_RESOURCE
	* Use reserved space on ext2 filesystems.
	* make ioctl(2) calls controlling ext3 journaling.
	* override disk quota limits.
	* increase resource limits (see setrlimit(2)).
	* override RLIMIT_NPROC resource limit.
	* override maximum number of consoles on console allocation.
	* override maximum number of keymaps.
	* allow more than 64hz interrupts from the real-time clock.
	* raise msg_qbytes limit for a System V message queue above the limit in /proc/sys/kernel/msgmnb (see msgop(2) and
	  msgctl(2)).
	* override the /proc/sys/fs/pipe-size-max limit when setting the capacity of a pipe using the F_SETPIPE_SZ fcntl(2)
	  command.
	* use F_SETPIPE_SZ to increase the capacity of a pipe above the limit specified by /proc/sys/fs/pipe-max-size.
	* override /proc/sys/fs/mqueue/queues_max limit when creating POSIX message queues (see mq_overview(7)).
	* employ prctl(2) PR_SET_MM operation.
	* set /proc/PID/oom_score_adj to a value lower than the value last set by a process with CAP_SYS_RESOURCE.`,
	}

	capSysTime = &capability{
		Name:  "CAP_SYS_TIME",
		Value: 25,
		Description: `CAP_SYS_TIME
	Set system clock (settimeofday(2), stime(2), adjtimex(2)); set real-time (hardware) clock.`,
	}

	capSysTtyConfig = &capability{
		Name:  "CAP_SYS_TTY_CONFIG",
		Value: 26,
		Description: `CAP_SYS_TTY_CONFIG
	Use vhangup(2); employ various privileged ioctl(2) operations on virtual terminals.`,
	}

	capMknod = &capability{
		Name:  "CAP_MKNOD",
		Value: 27,
		Description: `CAP_SYS_MKNOD (since Linux 2.4)
	Create special files using mknod(2).`,
	}

	capLease = &capability{
		Name:  "CAP_LEASE",
		Value: 28,
		Description: `CAP_LEASE (since Linux 2.4)
	Establish leases on arbitrary files (see fcntl(2)).`,
	}

	capAuditWrite = &capability{
		Name:  "CAP_AUDIT_WRITE",
		Value: 29,
		Description: `CAP_AUDIT_WRITE (since Linux 2.6.11)
	Write records to kernel auditing log.`,
	}

	capAuditControl = &capability{
		Name:  "CAP_AUDIT_CONTROL",
		Value: 30,
		Description: `CAP_AUDIT_CONTROL (since Linux 2.6.11)
	Enable and disable kernel auditing; change auditing filter rules; retrieve auditing status and filtering rules.`,
	}

	capSetfcap = &capability{
		Name:  "CAP_SETFCAP",
		Value: 31,
		Description: `CAP_SETFCAP (since Linux 2.6.24)
	Set file capabilities.`,
	}

	capMacOverride = &capability{
		Name:  "CAP_MAC_OVERRIDE",
		Value: 32,
		Description: `CAP_MAC_OVERRIDE (since Linux 2.6.25)
	Allow MAC configuration or state changes. Implemented for the Smack LSM.`,
	}

	capMacAdmin = &capability{
		Name:  "CAP_MAC_ADMIN",
		Value: 33,
		Description: `CAP_MAC_ADMIN (since Linux 2.6.25)
	Override Mandatory Access Control (MAC). Implemented for the Smack Linux Security Module (LSM).`,
	}

	capSyslog = &capability{
		Name:  "CAP_SYSLOG",
		Value: 34,
		Description: `CAP_SYSLOG (since Linux 2.6.37)
	* Perform privileged syslog(2) operations. See syslog(2) for information on which operations require privilege.
	* View kernel addresses exposed via /proc and other interfaces when /proc/sys/kernel/kptr_restrict has the value 1.
	  (See the discussion of the kptr_restrict in proc(5).)`,
	}

	capWakeAlarm = &capability{
		Name:  "CAP_WAKE_ALARM",
		Value: 35,
		Description: `CAP_WAKE_ALARM (since Linux 3.0)
	Trigger something that will wake up the system (set CLOCK_REALTIME_ALARM and CLOCK_BOOTTIME_ALARM timers).`,
	}

	capBlockSuspend = &capability{
		Name:  "CAP_WAKE_ALARM",
		Value: 36,
		Description: `CAP_BLOCK_SUSPEND (since Linux 3.5)
	Employ features that can block system suspend (epoll(7) EPOLLWAKEUP, /proc/sys/wake_lock).`,
	}

	capAuditRead = &capability{
		Name:  "CAP_AUDIT_READ",
		Value: 37,
		Description: `CAP_AUDIT_READ (since Linux 3.16)
	Allow reading the audit log via a multicast netlink socket.`,
	}
)

// Map maps each capability name to a struct with details about the capability.
var Map = map[string]*capability{
	"CAP_CHOWN":            capChown,
	"CAP_DAC_OVERRIDE":     capDacOverride,
	"CAP_DAC_READ_SEARCH":  capDacReadSearch,
	"CAP_FOWNER":           capFowner,
	"CAP_FSETID":           capFsetid,
	"CAP_KILL":             capKill,
	"CAP_SETGID":           capSetgid,
	"CAP_SETUID":           capSetuid,
	"CAP_SETPCAP":          capSetpcap,
	"CAP_LINUX_IMMUTABLE":  capLinuxImmutable,
	"CAP_NET_BIND_SERVICE": capNetBindService,
	"CAP_NET_BROADCAST":    capNetBroadcast,
	"CAP_NET_ADMIN":        capNetAdmin,
	"CAP_NET_RAW":          capNetRaw,
	"CAP_IPC_LOCK":         capIpcLock,
	"CAP_IPC_OWNER":        capIpcOwner,
	"CAP_SYS_MODULE":       capSysModule,
	"CAP_SYS_RAWIO":        capSysRawio,
	"CAP_SYS_CHROOT":       capSysChroot,
	"CAP_SYS_PTRACE":       capSysPtrace,
	"CAP_SYS_PACCT":        capSysPacct,
	"CAP_SYS_ADMIN":        capSysAdmin,
	"CAP_SYS_BOOT":         capSysBoot,
	"CAP_SYS_NICE":         capSysNice,
	"CAP_SYS_RESOURCE":     capSysResource,
	"CAP_SYS_TIME":         capSysTime,
	"CAP_SYS_TTY_CONFIG":   capSysTtyConfig,
	"CAP_MKNOD":            capMknod,
	"CAP_LEASE":            capLease,
	"CAP_AUDIT_WRITE":      capAuditWrite,
	"CAP_AUDIT_CONTROL":    capAuditControl,
	"CAP_SETFCAP":          capSetfcap,
	"CAP_MAC_OVERRIDE":     capMacOverride,
	"CAP_MAC_ADMIN":        capMacAdmin,
	"CAP_SYSLOG":           capSyslog,
	"CAP_WAKE_ALARM":       capWakeAlarm,
	"CAP_BLOCK_SUSPEND":    capBlockSuspend,
	"CAP_AUDIT_READ":       capAuditRead,
}

// Normalize takes a slice of capabilities, normalizes and unwraps CAP_ALL.
// The return values are a two slices: normalized capabilities slice that
// are valid and a slice with unrecognized capabilities.
func Normalize(capabilities []string) ([]string, []string) {
	const capAll = "CAP_ALL"

	capabilities = normalize(capabilities)

	// nolint:prealloc
	var included []string
	var excluded []string
	for _, capb := range capabilities {
		if capb == capAll {
			// do not reallocate memory if already did
			// this will NOT panic in case of nil slice
			excluded = excluded[:0]
			included = included[:0]
			for capb := range Map {
				included = append(included, capb)
			}
			break
		}
		if _, ok := Map[capb]; !ok {
			excluded = append(excluded, capb)
			continue
		}
		included = append(included, capb)
	}

	return RemoveDuplicated(included), RemoveDuplicated(excluded)
}

// Split takes a list of capabilities separated by commas and
// returns a string list with normalized capability name and a
// second list with unrecognized capabilities.
func Split(caps string) ([]string, []string) {
	if caps == "" {
		return []string{}, []string{}
	}
	return Normalize(strings.Split(caps, ","))
}

// RemoveDuplicated removes duplicated capabilities from provided list.
// It does not make copy of a passed list.
func RemoveDuplicated(caps []string) []string {
	for i := 0; i < len(caps); i++ {
		for j := i + 1; j < len(caps); j++ {
			if caps[i] == caps[j] {
				caps[j] = caps[len(caps)-1]
				caps = caps[:len(caps)-1]
				j--
			}
		}
	}
	return caps
}

func normalize(capabilities []string) []string {
	const capPrefix = "CAP_"
	for i, capb := range capabilities {
		capb = strings.TrimSpace(capb)
		capb = strings.ToUpper(capb)
		if !strings.HasPrefix(capb, capPrefix) {
			capb = capPrefix + capb
		}
		capabilities[i] = capb
	}
	return capabilities
}
