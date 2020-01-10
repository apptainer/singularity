// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

var ExecScript = `#!/bin/sh

for script in /.singularity.d/env/*.sh; do
    if [ -f "$script" ]; then
        . "$script"
    fi
done

# See https://github.com/sylabs/singularity/issues/2721,
# as bash is often used as the current shell it may confuse
# users if the provided command is /bin/bash implying to
# override PS1 set by singularity, then we may end up
# with a shell prompt identical to the host one, so we
# force PS1 through bash PROMPT_COMMAND
if test -z "${PROMPT_COMMAND:-}"; then
    export PROMPT_COMMAND="PS1=\"${PS1}\"; unset PROMPT_COMMAND"
else
    export PROMPT_COMMAND="${PROMPT_COMMAND:-}; PROMPT_COMMAND=\"\${PROMPT_COMMAND%%; PROMPT_COMMAND=*}\"; PS1=\"${PS1}\""
fi

exec "$@"
`

var ShellScript = `#!/bin/sh

for script in /.singularity.d/env/*.sh; do
    if [ -f "$script" ]; then
        . "$script"
    fi
done

# See https://github.com/sylabs/singularity/issues/2721,
# as bash is often used as the current shell it may confuse
# users when entering in singularity container via -s /bin/bash
# implying to override PS1 set by singularity and we may end up
# with a shell prompt identical to the host one, so we force PS1
# through bash PROMPT_COMMAND
if test -z "${PROMPT_COMMAND:-}"; then
    export PROMPT_COMMAND="PS1=\"${PS1}\"; unset PROMPT_COMMAND"
else
    export PROMPT_COMMAND="${PROMPT_COMMAND:-}; PROMPT_COMMAND=\"\${PROMPT_COMMAND%%; PROMPT_COMMAND=*}\"; PS1=\"${PS1}\""
fi

if test -n "$SINGULARITY_SHELL" -a -x "$SINGULARITY_SHELL"; then
    exec $SINGULARITY_SHELL "$@"

    echo "ERROR: Failed running shell as defined by '\$SINGULARITY_SHELL'" 1>&2
    exit 1

elif test -x /bin/bash; then
    SHELL=/bin/bash
    PS1="Singularity $SINGULARITY_NAME:\\w> "
    export SHELL PS1
    exec /bin/bash --norc "$@"
elif test -x /bin/sh; then
    SHELL=/bin/sh
    export SHELL
    exec /bin/sh "$@"
else
    echo "ERROR: /bin/sh does not exist in container" 1>&2
fi
exit 1
`

var RunScript = `#!/bin/sh

for script in /.singularity.d/env/*.sh; do
    if [ -f "$script" ]; then
        . "$script"
    fi
done

# See https://github.com/sylabs/singularity/issues/2721,
# if the runscript execute bash by default, it can give
# a prompt identical to the host one which may confuse users
# so we use PROMPT_COMMAND environment variable which is
# used by bash to execute a command before command prompt
# in order to set PS1 correctly before the first prompt
if test -z "${PROMPT_COMMAND:-}"; then
    export PROMPT_COMMAND="PS1=\"${PS1}\"; unset PROMPT_COMMAND"
else
    export PROMPT_COMMAND="${PROMPT_COMMAND:-}; PROMPT_COMMAND=\"\${PROMPT_COMMAND%%; PROMPT_COMMAND=*}\"; PS1=\"${PS1}\""
fi

if test -n "${SINGULARITY_APPNAME:-}"; then

    if test -x "/scif/apps/${SINGULARITY_APPNAME:-}/scif/runscript"; then
        exec "/scif/apps/${SINGULARITY_APPNAME:-}/scif/runscript" "$@"
    else
        echo "No runscript for contained app: ${SINGULARITY_APPNAME:-}"
        exit 1
    fi

elif test -x "/.singularity.d/runscript"; then
    exec "/.singularity.d/runscript" "$@"
else
    echo "No runscript found in container, executing /bin/sh"
    exec /bin/sh "$@"
fi
`

var TestScript = `#!/bin/sh

for script in /.singularity.d/env/*.sh; do
    if [ -f "$script" ]; then
        . "$script"
    fi
done


if test -n "${SINGULARITY_APPNAME:-}"; then

    if test -x "/scif/apps/${SINGULARITY_APPNAME:-}/scif/test"; then
        exec "/scif/apps/${SINGULARITY_APPNAME:-}/scif/test" "$@"
    else
        echo "No tests for contained app: ${SINGULARITY_APPNAME:-}"
        exit 1
    fi
elif test -x "/.singularity.d/test"; then
    exec "/.singularity.d/test" "$@"
else
    echo "No test found in container, executing /bin/sh -c true"
    exec /bin/sh -c true
fi
`

var StartScript = `#!/bin/sh

for script in /.singularity.d/env/*.sh; do
    if [ -f "$script" ]; then
        . "$script"
    fi
done

if test -x "/.singularity.d/startscript"; then
    exec "/.singularity.d/startscript" "$@"
fi
`
