# 
# Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
# 
# This software is licensed under a customized 3-clause BSD license.  Please
# consult LICENSE file distributed with the sources of this project regarding
# your rights to use or distribute this software.
# 
# 

if [ -n "${SING_USER_DEFINED_PREPEND_PATH:-}" ]; then
    PATH="${SING_USER_DEFINED_PREPEND_PATH}:${PATH}"
fi

if [ -n "${SING_USER_DEFINED_APPEND_PATH:-}" ]; then
    PATH="${PATH}:${SING_USER_DEFINED_APPEND_PATH}"
fi

if [ -n "${SING_USER_DEFINED_PATH:-}" ]; then
    PATH="${SING_USER_DEFINED_PATH}"
fi

unset SING_USER_DEFINED_PREPEND_PATH \
      SING_USER_DEFINED_APPEND_PATH \
      SING_USER_DEFINED_PATH

export PATH
