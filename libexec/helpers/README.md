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
A check should be a bash script that will perform some action. The following is required:

 - The script assumes a base of `SINGULARITY_ROOTFS`. This is defined when coming from bootstrap, and when called via [check.sh](check.sh), is set via `SINGULARITY_MOUNTPOINT`.
 - The script should indicate any warning/message to the user if the check is found to have failed. If pass (don't print anything? print pass?)
 - The script should return 0 when finished.
 - The script level, path, and tags should be added to [check.sh](check.sh) in the following format:

```
##################################################################################
# CHECK SCRIPTS
##################################################################################

#             [LEVEL] [SCRIPT]                                                            [TAGS]
execute_check 1       "$SINGULARITY_libexecdir/singularity/helpers/checks/1-hello-world.sh" security
execute_check 2       "$SINGULARITY_libexecdir/singularity/helpers/checks/2-hello-world.sh" security
execute_check 3       "$SINGULARITY_libexecdir/singularity/helpers/checks/3-hello-world.sh" security
```

The function `execute_check` will compare the level with the user specified (or default) `SINGULARITY_CHECKLEVEL` and execute the check only given it is under the specified threshold, and (not yet implemented) has the relevant tag. If no tag is specified, either security can be default, or performing all checks.


## Inspect
Inspect is called from [inspect.exec](../cli/inspect.exec), and serves to mount an image and then cat/return some subset of content from the singularity metadata folder (`.singularity.d`).
