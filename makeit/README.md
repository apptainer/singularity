# MAKEIT

## Goals

- To generate **native Makefiles** for the system *makeit* is running on.
  To accomplish that, we transform a set of Makefile fragments and
  module config files into a non-recursive Makefile that uses features
  available to all reasonable versions of Make (GNU, BSD, SVR4, etc).

- If **makeit** can be called a build system, it should stay so small and
  platform independent that it could be included in each project that
  it helps to build.

- To include/install and setup *makeit* for your project take a look at the
  INSTALL.md file.

## Module (\*.mconf) Keywords

- **name** : name of the module, just a handle
- **prog** : name of a program to link
- **lib** : name of a library to create, without the **lib** prefix
- **data** : name of a data file (symbols, pictures, text, etc.) to embed
- **asrc** : list of (.S) assembly source files
- **csrc** : list of C source files to build
- **win_asrc** : windows only list of C source files to build
- **win_csrc** : windows only list of C source files to build
- **unix_asrc** : unix only list of C source files to build
- **unix_csrc** : unix only list of C source files to build
- **depends** : list of module **name**'s that a prog or a lib depends on
- **cflags** : list of CFLAGS to add for this module
- **ldflags** : list of LDFLAGS to add for this module
- **extralibs** : list of extra libs needed by the program (e.g., -lgcc)
- **cleanfiles** : list of extra files to remove when *make clean* is called

## Implementation

- POSIX portable tools mainly awk and sh with system commands
