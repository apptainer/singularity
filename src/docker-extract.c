#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>
#include <sys/file.h>
#include <sys/param.h>
#include <sys/stat.h>
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
    static char buf[MAXPATHLEN];

    token = strrchr(opq_marker, '/');
    if (token == NULL) {
        singularity_message(ERROR, "Error getting dirname for opaque marker\n");
        ABORT(255);
    }

    retval = snprintf(buf, sizeof(buf), "%s/%s", rootfs_dir, opq_marker);
    if (retval == -1 || retval >= sizeof(buf)) {
        singularity_message(ERROR, "Error with pathname too long\n");
        ABORT(255);
    }

    if (is_dir(buf) == 0) {
        s_rmdir(buf);
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
    char buf[MAXPATHLEN];

    token = strstr(wh_marker, ".wh.");
    if (token == NULL) {
        singularity_message(ERROR, "Error getting filename for whiteout marker\n");
        ABORT(255);
    }

    // Start with ROOTFS
    retval = snprintf(buf, sizeof(buf), "%s/", rootfs_dir);
    if (retval == -1 || retval >= sizeof(buf)) {
        singularity_message(ERROR, "Error with pathname too long\n");
        ABORT(255);
    }
    l = strlen(buf);
    // Add whiteout path up to .wh.
    token_pos = strlen(wh_marker) - strlen(token);
    retval = snprintf(buf + l, token_pos + 1, "%s", wh_marker);
    if (retval == -1 || retval >= sizeof(buf) - l) {
        singularity_message(ERROR, "Error with pathname too long\n");
        ABORT(255);
    }
    l = strlen(buf);
    // Concatenate suffix after .wh
    retval = snprintf(buf + l, sizeof(buf) - l, "%s", token + 4);
    if (retval == -1 || retval >= sizeof(buf) - l) {
        singularity_message(ERROR, "Error with pathname too long\n");
        ABORT(255);
    }

    if (is_dir(buf) == 0) {
        retval = s_rmdir(buf);
    } else if (is_file(buf) == 0) {
        singularity_message(DEBUG, "Removing whiteout-ed file: %s\n",
                            buf);
        retval = unlink(buf);
    }

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

        if (strstr(pathname, "/.wh..wh..opq")) {
            singularity_message(DEBUG, "Opaque Marker %s\n", pathname);
            errcode = apply_opaque(pathname, rootfs_dir);
            if (errcode != 0) {
                break;
            }
        } else if (strstr(pathname, "/.wh.")) {
            singularity_message(DEBUG, "Whiteout Marker %s\n", pathname);
            errcode = apply_whiteout(pathname, rootfs_dir);
            if (errcode != 0) {
                break;
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

        // Do not extract whiteout markers (handled in apply_whiteouts)
        // Do not extract sockers, chr/blk devices, pipes
        if (strstr(pathname, "/.wh.") || pathtype == AE_IFSOCK ||
            pathtype == AE_IFCHR || pathtype == AE_IFBLK || pathtype == AE_IFIFO) {
            continue;
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
    char *tarfile = NULL;

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

    tarfile = argv[1];

    if (is_file(tarfile) < 0) {
        singularity_message(ERROR, "tar file does not exist: %s\n", tarfile);
        ABORT(255);
    }

    singularity_message(DEBUG, "Applying whiteouts for tar file %s\n", tarfile);
    retval = apply_whiteouts(tarfile, rootfs_dir);

    if (retval != 0) {
        singularity_message(ERROR, "Error applying layer whiteouts\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Extracting docker tar file %s\n", tarfile);
    retval = extract_tar(tarfile, rootfs_dir);

    if (retval != 0) {
        singularity_message(ERROR, "Error extracting tar file\n");
        ABORT(255);
    }

    return (retval);
}
