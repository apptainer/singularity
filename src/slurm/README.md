
Singularity plugin for SLURM
============================

This plugin allows users to execute their SLURM jobs within a Singularity container without
having to execute Singularity directly.  This assists in simplifying the invocation of the
container and hiding the implementation details.

To enable the plugin, add the following line to the SLURM plugin configuration (`/etc/slurm/plugstack.conf`):

```
required singularity.so
```

This works if Singularity is installed as a system package.  If the install prefix is `/opt/singularity`, then
one would have:

```
required /opt/singularity/lib/slurm/singularity.so
```

Note that the sysadmin may provide a default image that will be utilized if the user doesn't provide one:

```
required singularity.so default_image=/cvmfs/cernvm-prod.cern.ch/cvm3
```

Finally, a user may select their image through the `--singularity-image` optional argument:

```
srun --singularity-image=/cvmfs/cms.cern.ch/rootfs/x86_64/centos7/latest ls -lh /
```

Within a batch file, you would append this header:

```
#SBATCH --singularity-image=/cvmfs/cms.cern.ch/rootfs/x86_64/centos7/latest
```

