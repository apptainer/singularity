#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>
#include <sys/file.h>
#include <sys/stat.h>
#include <limits.h>
#include <locale.h>
#include <libgen.h>
#include <archive.h>
#include <archive_entry.h>

#include "config.h"
#include "util/file.h"
#include "util/message.h"
#include "util/registry.h"
#include "util/util.h"

/* apply_opaque
 *  Given opq_marker as a path to a whiteout opaque marker
 *    e.g. usr/share/doc/test/.wh..wh..opq
 *  Make the directory containing the make opaque for this layer by removing it
 *  if it exists under rootfs_dir
 */
int apply_opaque(const char *opq_marker, char *rootfs_dir) {
    int retval = 0;
    char *token;
    char target[PATH_MAX];
    char *target_real;

    token = strrchr(opq_marker, '/');
    if (token == NULL) {
        singularity_message(ERROR, "Error getting dirname for opaque marker\n");
        ABORT(255);
    }

    retval = snprintf(target, sizeof(target), "%s/%s", rootfs_dir, opq_marker);
    if (retval == -1 || retval >= sizeof(target)) {
        singularity_message(ERROR, "Error with pathname too long\n");
        ABORT(255);
    }

    // Target may not exist - that's ok
    retval = 0;

    if (is_dir(target) == 0) {

        target_real = realpath(target, NULL);  // Flawfinder: ignore

        if(target_real == NULL) {
            singularity_message(ERROR, "Error canonicalizing whiteout path %s - aborting.\n", target);
            ABORT(255);
        }

        if(strncmp(rootfs_dir, target_real, strlen(rootfs_dir)) != 0) {
            singularity_message(ERROR, "Attempt to whiteout outside of rootfs %s - aborting.\n", target_real);
            ABORT(255);
        }

        retval = s_rmdir(target_real);

        free(target_real);
    }


    return retval;
}

/* apply_whiteout
 *  Given wh_marker as a path to a whiteout marker
 *    e.g. usr/share/doc/test/.wh.deletedfile
 *  Whiteout the referenced file for this layer by removing it if it exists
 *  under rootfs_dir
 */
int apply_whiteout(const char *wh_marker, char *rootfs_dir) {
    int retval = 0;
    char* token;
    size_t token_pos, l = 0;
    char target[PATH_MAX];
    char *target_real;
    struct stat statbuf;

    token = strstr(wh_marker, ".wh.");
    if (token == NULL) {
        singularity_message(ERROR, "Error getting filename for whiteout marker\n");
        ABORT(255);
    }

    // Start with ROOTFS
    retval = snprintf(target, sizeof(target), "%s/", rootfs_dir);
    if (retval == -1 || retval >= sizeof(target)) {
        singularity_message(ERROR, "Error with pathname too long\n");
        ABORT(255);
    }
    l = strlen(target);
    // Add whiteout path up to .wh.
    if (strlen(target) + strlen(token) > sizeof(target) - 1) {
        singularity_message(ERROR, "Error with pathname too long\n");
        ABORT(255);
    }
    token_pos = strlen(wh_marker) - strlen(token);
    retval = snprintf(target + l, token_pos + 1, "%s", wh_marker);
    if (retval == -1 || retval >= sizeof(target) - l) {
        singularity_message(ERROR, "Error with pathname too long\n");
        ABORT(255);
    }
    l = strlen(target);
    // Concatenate suffix after .wh
    retval = snprintf(target + l, sizeof(target) - l, "%s", token + 4);
    if (retval == -1 || retval >= sizeof(target) - l) {
        singularity_message(ERROR, "Error with pathname too long\n");
        ABORT(255);
    }

    // Target may not exist - that's ok
    if (stat(target, &statbuf) < 0){
        singularity_message(DEBUG, "Whiteout target doesn't exist, at: %s\n",
                            target);

        return 0;
    }

    // If the whiteout target is a link, we need to remove that link (source)
    // itself and not resolve through it
    if(is_link(target) == 0) {
        char *target_copy1, *target_copy2, *parent, *link, *parent_real;

        target_copy1 = strdup(target);
        if(target_copy1 == NULL) {
            singularity_message(ERROR, "Error allocating memory for path - aborting.\n");
            ABORT(255);
        }

        target_copy2 = strdup(target);
        if(target_copy2 == NULL) {
            singularity_message(ERROR, "Error allocating memory for path - aborting.\n");
            ABORT(255);
        }

        parent = dirname(target_copy1);
        link = basename(target_copy2);

        singularity_message(DEBUG, "Whiteout target is a symlink with parent dir: %s Link: %s\n",
                            parent, link);


        // First check fully resolved *parent dir* does not escape the ROOTFS
        parent_real = realpath(parent, NULL);   // Flawfinder: ignore
        if(parent_real == NULL) {
            singularity_message(ERROR, "Error canonicalizing whiteout path %s - aborting.\n", target);
            ABORT(255);
        }


        singularity_message(DEBUG, "Link parent dir resolves to: %s\n",
                            parent_real);


        if(strncmp(rootfs_dir, parent_real, strlen(rootfs_dir)) != 0) {
            singularity_message(ERROR, "Attempt to whiteout outside of rootfs %s - aborting.\n", parent_real);
            ABORT(255);
        }

        // And the link cannot be called '..' (not sure if this is possible, but
        // maybe a tar can be created like that maliciously)
        if(strncmp(link, "..", sizeof(link) -1) == 0){
            singularity_message(ERROR, "Whiteout target has '..' as last component: %s - aborting.\n", target);
            ABORT(255);
        }

        // Now our real target path is the resolved parent, plus the link
        // basename of the link
        target_real = malloc(PATH_MAX + 1);
        retval = snprintf(target_real, PATH_MAX - 1, "%s/%s", parent_real, link );
        if (retval == -1 || retval >= PATH_MAX - 1) {
            singularity_message(ERROR, "Error with pathname too long\n");
            ABORT(255);
        }

        singularity_message(DEBUG, "Whiteout target resolves to symlink at: %s\n",
                            target_real);

        free(target_copy1);
        free(target_copy2);
        free(parent_real);

    }else{

        target_real = realpath(target, NULL);  // Flawfinder: ignore

        if(target_real == NULL) {
            singularity_message(ERROR, "Error canonicalizing whiteout path %s - aborting.\n", target);
            ABORT(255);
        }

        if(strncmp(rootfs_dir, target_real, strlen(rootfs_dir)) != 0) {
            singularity_message(ERROR, "Attempt to whiteout outside of rootfs %s - aborting.\n", target_real);
            ABORT(255);
        }

        singularity_message(DEBUG, "Whiteout target is a regular file/dir, at: %s\n",
                            target_real);
    }


    if (is_dir(target_real) == 0) {
        retval = s_rmdir(target_real);
    } else if (is_file(target_real) == 0 || is_link(target_real) == 0) {
        singularity_message(DEBUG, "Removing whiteout-ed file: %s\n",
                            target_real);
        retval = unlink(target_real);
    }

    free(target_real);

    return retval;
}

/* apply_whiteouts
 *  Process tarfile and apply any aufs opaque/whiteouts on rootfs_dir
 */
int apply_whiteouts(char *tarfile, char *rootfs_dir) {
    int retval = 0;
    int errcode = 0;
    struct archive *a;
    struct archive_entry *entry;

    a = archive_read_new();
#if ARCHIVE_VERSION_NUMBER <= 3000000
    archive_read_support_compression_all(a);
#else
    archive_read_support_filter_all(a);
#endif
    archive_read_support_format_all(a);
    retval = archive_read_open_filename(a, tarfile, 10240);
    if (retval != ARCHIVE_OK) {
        return (1);
    }

    while (archive_read_next_header(a, &entry) == ARCHIVE_OK) {

        const char *pathname = archive_entry_pathname(entry);

        if (*pathname == '/') {
            singularity_message(ERROR, "Archive contains absolute paths %s - aborting.\n", pathname);
            ABORT(255);
        }

        if (strstr(pathname, "/.wh..wh..opq")) {
            singularity_message(DEBUG, "Opaque Marker %s\n", pathname);
            errcode = apply_opaque(pathname, rootfs_dir);
            if (errcode != 0) {
                singularity_message(ERROR, "Error applying opaque marker from docker layer.\n");
                ABORT(255);
            }
        } else if (strstr(pathname, "/.wh.")) {
            singularity_message(DEBUG, "Whiteout Marker %s\n", pathname);
            errcode = apply_whiteout(pathname, rootfs_dir);
            if (errcode != 0) {
                singularity_message(ERROR, "Error applying whiteout marker from docker layer.\n");
                ABORT(255);
            }
        }
    }
#if ARCHIVE_VERSION_NUMBER <= 3000000
    retval = archive_read_finish(a);
#else
    retval = archive_read_free(a);
#endif
    if (retval != ARCHIVE_OK){
        singularity_message(ERROR, "Error freeing archive\n");
        ABORT(255);
    }

    return errcode;
}

/* See  https://github.com/libarchive/libarchive/wiki/Examples#A_Complete_Extractor */
static int copy_data(struct archive *ar, struct archive *aw) {
    int r;
    const void *buff;
    size_t size;
    int64_t offset;

    for (;;) {
        r = archive_read_data_block(ar, &buff, &size, &offset);
        if (r == ARCHIVE_EOF) {
            return (ARCHIVE_OK);
        }
        if (r < ARCHIVE_OK) {
            return (r);
        }
        r = archive_write_data_block(aw, buff, size, offset);
        if (r < ARCHIVE_OK) {
            singularity_message(ERROR, "tar extraction error: %s\n", archive_error_string(aw));
            return (r);
        }
    }
}

/* extract_tar
 *  Extract a tar file to rootfs_dir using libarchive. Handles compression.
 *  Exclude any .wh. whiteout files, and device/pipe/fifo entries
 *
 * See https://github.com/libarchive/libarchive/wiki/Examples#A_Complete_Extractor
 */
int extract_tar(const char *tarfile, const char *rootfs_dir) {
    int retval = 0;
    struct archive *a;
    struct archive *ext;
    struct archive_entry *entry;
    mode_t perms;
    int flags;
    int r;
    char *orig_dir;
    const char *pathname;
    int pathtype;

    orig_dir = get_current_dir_name();

    /* Select which attributes we want to restore. */
    flags = ARCHIVE_EXTRACT_TIME;
    flags |= ARCHIVE_EXTRACT_PERM;
    flags |= ARCHIVE_EXTRACT_ACL;
    flags |= ARCHIVE_EXTRACT_FFLAGS;

    flags |= ARCHIVE_EXTRACT_SECURE_SYMLINKS;
    flags |= ARCHIVE_EXTRACT_SECURE_NODOTDOT;

    a = archive_read_new();
    archive_read_support_format_all(a);
#if ARCHIVE_VERSION_NUMBER <= 3000000
    archive_read_support_compression_all(a);
#else
    archive_read_support_filter_all(a);
#endif
    ext = archive_write_disk_new();
    archive_write_disk_set_options(ext, flags);
    archive_write_disk_set_standard_lookup(ext);
    if ((r = archive_read_open_filename(a, tarfile, 10240))){
        singularity_message(ERROR, "Error opening tar file %s\n", tarfile);
        ABORT(255);
    }

    // Extract into the SINGULARITY_ROOTFS
    r = chdir(rootfs_dir);
    if (r < 0 ){
        singularity_message(ERROR, "Could not chdir to SINGULARITY_ROOTFS %s\n", rootfs_dir);
        ABORT(255);
    }

    for (;;) {
        r = archive_read_next_header(a, &entry);

        if (r == ARCHIVE_EOF) {
            break;
        }

        if (r < ARCHIVE_OK) {
            singularity_message(WARNING, "Warning reading tar header: %s\n", archive_error_string(a));
        }
        if (r < ARCHIVE_WARN) {
            singularity_message(ERROR, "Error reading tar header: %s\n", archive_error_string(a));
            ABORT(255);
        }

        pathname = archive_entry_pathname(entry);
        pathtype = archive_entry_filetype(entry);

        if (*pathname == '/') {
            singularity_message(ERROR, "Archive contains absolute paths - aborting.\n");
            ABORT(255);
        }

        // Do not extract whiteout markers (handled in apply_whiteouts)
        // Do not extract sockers, chr/blk devices, pipes
        if (strstr(pathname, "/.wh.") || pathtype == AE_IFSOCK ||
            pathtype == AE_IFCHR || pathtype == AE_IFBLK || pathtype == AE_IFIFO) {
            continue;
        }

        // Issue 977 - Force write perms needed for user builds
        if(getuid() != 0) {
#if ARCHIVE_VERSION_NUMBER <= 3000000
            perms = archive_entry_mode(entry);
            if( (perms & S_IWUSR) != S_IWUSR) {
                archive_entry_set_mode(entry, perms | S_IWUSR);
            }
#else
            perms = archive_entry_perm(entry);
            if( (perms & S_IWUSR) != S_IWUSR) {
                archive_entry_set_perm(entry, perms | S_IWUSR);
            }
#endif
        }

        r = archive_write_header(ext, entry);
        if (r < ARCHIVE_OK) {
            singularity_message(WARNING, "Warning handling tar header: %s\n", archive_error_string(ext));
        }else if (archive_entry_size(entry) > 0) {
            r = copy_data(a, ext);
            if (r < ARCHIVE_OK) {
                singularity_message(WARNING, "Warning handling tar header: %s\n", archive_error_string(ext));
            }
            if (r < ARCHIVE_WARN) {
                singularity_message(ERROR, "Error handling tar header: %s\n", archive_error_string(ext));
                ABORT(255);
            }
        }
        r = archive_write_finish_entry(ext);
        if (r < ARCHIVE_OK) {
            singularity_message(WARNING, "Warning freeing archive entry: %s\n", archive_error_string(ext));
        }
        if (r < ARCHIVE_WARN) {
            singularity_message(ERROR, "Error freeing archive entry: %s\n", archive_error_string(ext));
            ABORT(255);
        }
    }
    archive_read_close(a);
#if ARCHIVE_VERSION_NUMBER <= 3000000
    archive_read_finish(a);
    archive_write_close(ext);
    archive_write_finish(ext);
#else
    archive_read_free(a);
    archive_write_close(ext);
    archive_write_free(ext);
#endif
    r = chdir(orig_dir);
    if (r < 0 ){
        singularity_message(ERROR, "Could not chdir back to %s\n", orig_dir);
        ABORT(255);
    }

    free(orig_dir);

    return (retval);
}

int main(int argc, char **argv) {
    int retval = 0;
    char *rootfs_dir = singularity_registry_get("ROOTFS");
    char *rootfs_realpath;
    char *tarfile = NULL;

    // Set UTF8 locale so that libarchive doesn't produce warnings for UTF8
    // names - en_US.UTF-8 is most likely to be available
    if( setlocale(LC_ALL, "en_US.UTF-8") == NULL ) {
        // Fall back to C.UTF-8 for super-minimal debian and musl based distros
        if (setlocale(LC_ALL, "C.UTF-8") == NULL ) {
            singularity_message(WARNING, "Could not set a UTF8 locale, layer extraction may produce warnings\n");
        }
    }

    if (argc != 2) {
        singularity_message(ERROR, "Provide a single docker tar file to extract\n");
        ABORT(255);
    }

    if (rootfs_dir == NULL) {
        singularity_message(ERROR, "Environment is not properly setup\n");
        ABORT(255);
    }

    if (is_dir(rootfs_dir) < 0) {
        singularity_message(ERROR, "SINGULARITY_ROOTFS does not exist\n");
        ABORT(255);
    }

    rootfs_realpath = realpath(rootfs_dir, NULL);  // Flawfinder: ignore

    if(rootfs_realpath == NULL) {
        singularity_message(ERROR, "Error canonicalizing ROOTFS path %s - aborting.\n", rootfs_dir);
        ABORT(255);
    }

    singularity_message(DEBUG, "ROOTFS %s canonicalized to %s\n", rootfs_dir, rootfs_realpath);

    if( strlen(rootfs_realpath) == 1 && *rootfs_realpath == '/') {
        singularity_message(ERROR, "Refusing to extract into host root / - aborting.\n");
        ABORT(255);
    }

    tarfile = argv[1];

    if (is_file(tarfile) < 0) {
        singularity_message(ERROR, "tar file does not exist: %s\n", tarfile);
        ABORT(255);
    }

    singularity_message(DEBUG, "Applying whiteouts for tar file %s\n", tarfile);
    retval = apply_whiteouts(tarfile, rootfs_realpath);

    if (retval != 0) {
        singularity_message(ERROR, "Error applying layer whiteouts\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Extracting docker tar file %s\n", tarfile);
    retval = extract_tar(tarfile, rootfs_realpath);

    if (retval != 0) {
        singularity_message(ERROR, "Error extracting tar file\n");
        ABORT(255);
    }

    free(rootfs_realpath);

    return (retval);
}
