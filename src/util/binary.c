/* 
 * Copyright (c) 2017, EDF, SA. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 */


#define _GNU_SOURCE
#include <stdio.h>
#include <errno.h>
#include <libelf.h>

#include "util/message.h"
#include "util/binary.h"

int singularity_binary_arch(char* path){
    FILE* fp;
    Elf* elf_file;
    Elf32_Ehdr* hdr32 = NULL;
    Elf64_Ehdr* hdr64 = NULL;

    fp = fopen(path, "rb");
    if (fp == NULL) {
        singularity_message(WARNING, "Failed to open binary: %s (error=%d)\n", path, errno);
        return(BINARY_ARCH_UNKNOWN);
    }
    if (elf_version(EV_CURRENT) == EV_NONE) {
        singularity_message(WARNING, "Failed to initialize ELF library: %s\n", elf_errmsg(-1));
        return(BINARY_ARCH_UNKNOWN);
    }
    elf_file = elf_begin(fileno(fp), ELF_C_READ, NULL);
    if (elf_file == NULL) {
        singularity_message(DEBUG, "Failed initialize elf parsing on file %s: %s\n", path, elf_errmsg(-1));
        // Not an ELF BINARY
        return(BINARY_ARCH_UNKNOWN);
    }
    hdr32 = elf32_getehdr(elf_file);
    if (hdr32 == NULL) {
        hdr64 = elf64_getehdr(elf_file);
    }
    fclose(fp);
    if (hdr32 == NULL) {
        if (hdr64 == NULL) {
            singularity_message(DEBUG, "No ELF headers on binary file %s: (%s)\n", path, elf_errmsg(-1));
            // Not an ELF BINARY
            return(BINARY_ARCH_UNKNOWN);
        } else {
            if (hdr64->e_machine == EM_X86_64) {
                // x86_64
                return(BINARY_ARCH_X86_64);
            } else {
                // Unkown 64 bits arch
                return(BINARY_ARCH_UNKNOWN);
            }
        }
    } else {
        switch(hdr32->e_machine) {
            case EM_X86_64 :
                // x32
                return(BINARY_ARCH_X32);
            case EM_386 : 
                // i386
                return(BINARY_ARCH_I386);
            default :
                // Unkown 32 bits arch
                return(BINARY_ARCH_UNKNOWN);
        }
    }
}
