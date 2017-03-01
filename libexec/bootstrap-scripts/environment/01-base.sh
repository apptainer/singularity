
if test -z "$SINGULARITY_INIT"; then
    PATH=$PATH:/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin
    PS1="Singularity $SINGULARITY_CONTAINER> "
    LD_LIBRARY_PATH="$LD_LIBRARY_PATH:/usr/local/lib:/usr/local/lib64"
    SINGULARITY_INIT=1
    export PATH PS1 SINGULARITY_INIT LD_LIBRARY_PATH
fi
