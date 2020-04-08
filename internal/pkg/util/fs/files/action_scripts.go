// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package files

var ActionScript = `#!/bin/sh

declare -r __exported_env__=$(getallenv)
declare -r __singularity_cmd__=${SINGULARITY_COMMAND:-}

if test -n "${SINGULARITY_APPNAME:-}"; then
    readonly SINGULARITY_APPNAME
fi

export PWD

clear_env() {
    local IFS=$'\n'

    for e in ${__exported_env__}; do
        key=$(getenvkey "${e}")
        case "${key}" in
        PWD|HOME|OPTIND|UID|SINGULARITY_APPNAME|SINGULARITY_SHELL)
            ;;
        SINGULARITY_NAME|SINGULARITY_CONTAINER)
            readonly "${key}"
            ;;
        *)
            unset "${key}"
            ;;
        esac
    done
}

restore_env() {
    local IFS=$'\n'

    # restore environment variables which haven't been
    # defined by docker or virtual file above, empty
    # variables are also unset
    for e in ${__exported_env__}; do
        key=$(getenvkey "${e}")
        if ! test -v "${key}"; then
            export "${e}"
        elif test -z "${!key}"; then
            unset "${key}"
        fi
    done
}

clear_env

if test -d "/.singularity.d/env"; then
    for __script__ in /.singularity.d/env/*.sh; do
        if test -f "${__script__}"; then
            sylog debug "Sourcing ${__script__}"

            case "${__script__}" in
            /.singularity.d/env/90-environment.sh)
                # docker files below may not be present depending of image source
                # and build, so we also fix the PATH if not defined here
                if ! test -v PATH; then
                    export PATH="$(fixpath)"
                fi
                source "${__script__}"
                ;;
            /.singularity.d/env/10-docker2singularity.sh| \
            /.singularity.d/env/10-docker.sh)
                source "${__script__}"
                # append potential missing path from the default PATH
                # used by Singularity
                export PATH="$(fixpath)"
                ;;
            /.singularity.d/env/99-base.sh)
                # this file is the common denominator in image built since
                # Singularity 2.3, inject forwarded variables right after
                source "${__script__}"
                source "/.inject-singularity-env.sh"
                ;;
            *)
                source "${__script__}"
                ;;
            esac
        fi
    done
else
    # this is for old images built with Singularity version prior to 2.3
    if test -f "/environment"; then
        source "/environment"
        unset PATH
        export PATH="$(fixpath)"
    fi
    source "/.inject-singularity-env.sh"
fi

if ! test -f "/.singularity.d/env/99-runtimevars.sh"; then
    source "/.singularity.d/env/99-runtimevars.sh"
fi

restore_env

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

sylog debug "Running action command ${__singularity_cmd__}"

case "${__singularity_cmd__}" in
exec)
    exec "$@" ;;
shell)
    if test -n "${SINGULARITY_SHELL:-}" -a -x "${SINGULARITY_SHELL:-}"; then
        exec "${SINGULARITY_SHELL:-}" "$@"
    elif test -x "/bin/bash"; then
        export SHELL=/bin/bash
        exec "/bin/bash" --norc "$@"
    elif test -x "/bin/sh"; then
        export SHELL=/bin/sh
        exec "/bin/sh" "$@"
    fi

    sylog error "/bin/sh does not exist in container"
    exit 1 ;;
run)
    if test -n "${SINGULARITY_APPNAME:-}"; then
        if test -x "/scif/apps/${SINGULARITY_APPNAME:-}/scif/runscript"; then
            exec "/scif/apps/${SINGULARITY_APPNAME:-}/scif/runscript" "$@"
        fi
        sylog error "no runscript for contained app: ${SINGULARITY_APPNAME:-}"
        exit 1
    elif test -x "/.singularity.d/runscript"; then
        exec "/.singularity.d/runscript" "$@"
    elif test -x "/singularity"; then
        exec "/singularity" "$@"
    elif test -x "/bin/sh"; then
        sylog info "No runscript found in container, executing /bin/sh"
        exec "/bin/sh" "$@"
    fi

    sylog error "No runscript and no /bin/sh executable found in container, aborting"
    exit 1 ;;
test)
    if test -n "${SINGULARITY_APPNAME:-}"; then
        if test -x "/scif/apps/${SINGULARITY_APPNAME:-}/scif/test"; then
            exec "/scif/apps/${SINGULARITY_APPNAME:-}/scif/test" "$@"
        fi
        sylog error "No tests for contained app: ${SINGULARITY_APPNAME:-}"
        exit 1
    elif test -x "/.singularity.d/test"; then
        exec "/.singularity.d/test" "$@"
    fi

    sylog info "No test script found in container, exiting"
    exit 0 ;;
start)
    if test -x "/.singularity.d/startscript"; then
        exec "/.singularity.d/startscript" "$@"
    fi

    sylog info "No instance start script found in container"
    exit 0 ;;
*)
    sylog error "Unknown action ${__singularity_cmd__}"
    exit 1 ;;
esac
`

var RuntimeVars = `#!/bin/sh
if test -n "${SING_USER_DEFINED_PREPEND_PATH:-}"; then
    PATH="${SING_USER_DEFINED_PREPEND_PATH}:${PATH}"
    unset SING_USER_DEFINED_PREPEND_PATH
fi

if test -n "${SING_USER_DEFINED_APPEND_PATH:-}"; then
    PATH="${PATH}:${SING_USER_DEFINED_APPEND_PATH}"
    unset SING_USER_DEFINED_APPEND_PATH
fi

if test -n "${SING_USER_DEFINED_PATH:-}"; then
    PATH="${SING_USER_DEFINED_PATH}"
    unset SING_USER_DEFINED_PATH
fi

export PATH
`
