# Helpers

## Checks
Broadly, a check is a script that is run over a mounted filesystem, primary with the purpose of checking for some security issue. This process is tighly controlled, meaning that the script names in the [checks](checks) folder are hard coded into the script [check.sh](check.sh). The flow of checks is the following:

 - the user calls `singularity check container.img` to invoke [check.exec](../cli/check.exec)
 - specification of `--low` (3), `--med` (2), or `--high` (1) sets the level to perform. The level is a filter, meaning that a level of 3 will include 3,2,1, and a level of 1 (high) will only call checks of high priority.
 - specification of `--tag/--tags` will allow the user (or execution script) to specify a kind of check. This is primarily to allow for extending the checks to do other types of things. For example, for this initial batch, these are all considered `security` checks. However, we can also take multiple tags. A script might have tag `security` but also tag `debootstrap` to specify it should only perform checks with one or both these tags. The [check.help](../cli/check.help) displays examples of how the user makes this specification:

```
# Perform all security checks, these are the same
$ singularity check ubuntu.img
$ singularity check --tag security ubuntu.img

# Perform high level security checks
$ singularity check --high ubuntu.img

# All security checks for tag security AND debootstrap
$ singularity check --tag "security+debootstrap" ubuntu.img

# All security checks for tag security OR debootstrap
$ singularity check --tag "security|debootstrap" ubuntu.img
```
 
### Adding a Check
A check should be a bash (or other) script that will perform some action. The following is required:

**Relative to SINGULARITY_ROOTFS**
The script must perform check actions relative to `$SINGULARITY_ROOTFS`. For example, in python you might change directory to this location:

```
import os
base = os.environ["SINGULARITY_ROOTFS"]
os.chdir(base)
```

or do the same in bash:

```
cd $SINGULARITY_ROOTFS
ls $SINGULARITY_ROOTFS/var
```

Since we are doing a mount, all checks must be static relative to this base, otherwise you are likely checking the host system.

**Verbose**
The script should indicate any warning/message to the user if the check is found to have failed. If pass, the check's name and status will be printed, with any relevant information. For more thorough checking, you might want to give more verbose output.

**Return Code**
The script return code of "success" is defined in [check.sh](check.sh), and other return codes are considered not success. When a non success return code is found, the rest of the checks continue running, and no action is taken. We might want to give some admin an ability to specify a check, a level, and prevent continuation of the build/bootstrap given a fail.

**Check.sh**
The script level, path, and tags should be added to [check.sh](check.sh) in the following format:

```
##################################################################################
# CHECK SCRIPTS
##################################################################################

#        [SUCCESS] [LEVEL]  [SCRIPT]                                                                         [TAGS]
execute_check    0    HIGH  "bash $SINGULARITY_libexecdir/singularity/helpers/checks/1-hello-world.sh"       security
execute_check    0     LOW  "python $SINGULARITY_libexecdir/singularity/helpers/checks/2-cache-content.py"   clean
execute_check    0    HIGH  "python $SINGULARITY_libexecdir/singularity/helpers/checks/3-cve.py"             security
```

The function `execute_check` will compare the level (`[LEVEL]`) with the user specified (or default) `SINGULARITY_CHECKLEVEL` and execute the check only given it is under the specified threshold, and (not yet implemented) has the relevant tag. The success code is also set here with `[SUCCESS]`. Currently, we aren't doing anything with `[TAGS]` and thus perform all checks.


## Inspect
Inspect is called from [inspect.exec](../cli/inspect.exec), and serves to mount an image and then cat/return some subset of content from the singularity metadata folder (`.singularity.d`).
