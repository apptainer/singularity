#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>
#include <sys/file.h>
#include <sys/stat.h>
#include <archive.h>
#include <archive_entry.h>

#include "config.h"
#include "util/util.h"
#include "util/message.h"
#include "util/file.h"

int apply_opaque(char* pathname, char* rootfs_dir) {
    int retval = 0;
    char *token = strrchr(pathname, '/');

    if( token == NULL ){
        singularity_message(ERROR, "Error getting dirname for opaque marker\n");
        ABORT(255);
    }

    size_t length = token - pathname;
    char *opq_dir = malloc(length + 1);
    strncpy(opq_dir, pathname, length);
    opq_dir[length] = 0;

    char *opq_dir_rootfs = malloc(strlen(rootfs_dir) + 1 + strlen(opq_dir) + 1 );
    sprintf(opq_dir_rootfs, "%s/%s", rootfs_dir, opq_dir);

    if( is_dir(opq_dir_rootfs) == 0 ){
        s_rmdir(opq_dir_rootfs);
    }

    free(opq_dir);
    free(opq_dir_rootfs);

    return retval;
}


int apply_whiteout(char* pathname, char* rootfs_dir) {
    int retval = 0;
    char *token = strstr(pathname, ".wh.");

    if( token == NULL ){
        singularity_message(ERROR, "Error getting filename for whiteout marker\n");
        ABORT(255);
    }

    size_t token_pos = strlen(pathname) - strlen(token);
    size_t length = strlen(pathname) - strlen(".wh.") + 1;
    char *wht_path = malloc(length);
    strncpy(wht_path, pathname, token_pos + 1 );
    wht_path[token_pos] = 0;
    strcat(wht_path, token + 4);

    char *wht_path_rootfs = malloc(strlen(rootfs_dir) + 1 + strlen(wht_path) + 1 );
    sprintf(wht_path_rootfs, "%s/%s", rootfs_dir, wht_path);

    if( is_dir(wht_path_rootfs) == 0 ) {
        retval = s_rmdir(wht_path_rootfs);
    }else if(is_file(wht_path_rootfs) == 0 ) {
        singularity_message(DEBUG, "Removing whiteout-ed file: %s\n", wht_path_rootfs);
        retval = unlink(wht_path_rootfs);
    }

    free(wht_path);
    free(wht_path_rootfs);

    return retval;
}


int apply_whiteouts(char *tarfile, char *rootfs_dir) {
    int ret = 0;
    int errcode = 0;

    struct archive *a;
    struct archive_entry *entry;

    a = archive_read_new();
    archive_read_support_filter_all(a);
    archive_read_support_format_all(a);
    ret = archive_read_open_filename(a, tarfile, 10240);
    if (ret != ARCHIVE_OK)
        return(1);

    char *pathname;

    while (archive_read_next_header(a, &entry) == ARCHIVE_OK) {

        pathname = archive_entry_pathname(entry);

        if (strstr(pathname, "/.wh..wh..opq")){
            singularity_message(DEBUG, "Opaque Marker %s\n", pathname);
            errcode = apply_opaque(pathname, rootfs_dir);
            if ( errcode != 0) {
                break;
            }
        }else if (strstr(pathname, "/.wh.")){
            singularity_message(DEBUG, "Whiteout Marker %s\n", pathname);
            errcode = apply_whiteout(pathname, rootfs_dir);
            if ( errcode != 0) {
                break;
            }
        }

    }
    ret = archive_read_free(a);  // Note 3
    if (ret != ARCHIVE_OK)
        return(1);

    return errcode;

}


static int copy_data(struct archive *ar, struct archive *aw)
{
    int r;
    const void *buff;
    size_t size;
    int64_t offset;

    for (;;) {
        r = archive_read_data_block(ar, &buff, &size, &offset);
        if (r == ARCHIVE_EOF)
            return (ARCHIVE_OK);
        if (r < ARCHIVE_OK)
            return (r);
        r = archive_write_data_block(aw, buff, size, offset);
        if (r < ARCHIVE_OK) {
            fprintf(stderr, "%s\n", archive_error_string(aw));
            return (r);
        }
    }
}


int extract_tar(char* tarfile, char* rootfs_dir) {
    int retval = 0;
    struct archive *a;
    struct archive *ext;
    struct archive_entry *entry;
    int flags;
    int r;
    char *orig_dir;

    orig_dir = getcwd(orig_dir,  0);

    /* Select which attributes we want to restore. */
    flags = ARCHIVE_EXTRACT_TIME;
    flags |= ARCHIVE_EXTRACT_PERM;
    flags |= ARCHIVE_EXTRACT_ACL;
    flags |= ARCHIVE_EXTRACT_FFLAGS;


    a = archive_read_new();
    archive_read_support_format_all(a);
    archive_read_support_compression_all(a);
    ext = archive_write_disk_new();
    archive_write_disk_set_options(ext, flags);
    archive_write_disk_set_standard_lookup(ext);
    if ((r = archive_read_open_filename(a, tarfile, 10240)))
        exit(1);

    chdir(rootfs_dir);

    for (;;) {
        r = archive_read_next_header(a, &entry);

        if (r == ARCHIVE_EOF)
            break;

        if (r < ARCHIVE_OK)
            singularity_message(WARNING, "A%s\n", archive_error_string(a));
        if (r < ARCHIVE_WARN) {
            singularity_message(ERROR, "A%s\n", archive_error_string(a));
            ABORT(255);
        }

        char *pathname = archive_entry_pathname(entry);
        int pathtype = archive_entry_filetype(entry);

        if (strstr(pathname, "/.wh.") ||
            pathtype == AE_IFSOCK     ||
            pathtype == AE_IFCHR      ||
            pathtype == AE_IFBLK      ||
            pathtype == AE_IFIFO
        ) {
          continue;
        }

        r = archive_write_header(ext, entry);
        if (r < ARCHIVE_OK)
            singularity_message(WARNING, "%s\n", archive_error_string(ext));
        else if (archive_entry_size(entry) > 0) {
            r = copy_data(a, ext);
            if (r < ARCHIVE_OK)
                singularity_message(WARNING, "%s\n", archive_error_string(ext));
            if (r < ARCHIVE_WARN) {
                singularity_message(ERROR, "%Bs\n", archive_error_string(ext));
                ABORT(255);
            }
        }
        r = archive_write_finish_entry(ext);
        if (r < ARCHIVE_OK)
            singularity_message(WARNING, "%s\n", archive_error_string(ext));
        if (r < ARCHIVE_WARN) {
            singularity_message(ERROR, "%Cs\n", archive_error_string(ext));
            ABORT(255);
        }
    }
    archive_read_close(a);
    archive_read_free(a);
    archive_write_close(ext);
    archive_write_free(ext);

    chdir(orig_dir);
    free(orig_dir);

    return(retval);

}


int main(int argc, char **argv) {
    int retval = 0;
    char *rootfs_dir = envar_path("SINGULARITY_ROOTFS");
    char *tarfile = NULL;

    if ( rootfs_dir == NULL ) {
      singularity_message(ERROR, "Environment is not properly setup\n");
      ABORT(255);
    }

    if (is_dir(rootfs_dir) < 0 ){
        singularity_message(ERROR, "SINGULARITY_ROOTFS does not exist\n");
        ABORT(255);
    }

    if (argc != 2) {
        singularity_message(ERROR, "Provide a single docker tar file to extract\n");
        ABORT(255);
    }

    tarfile = argv[1];

    singularity_message(DEBUG, "Applying whiteouts for tar file %s\n", tarfile);

    retval = apply_whiteouts(tarfile, rootfs_dir);

    if ( retval != 0){
        singularity_message(ERROR, "Error applying layer whiteouts\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Extracting docker tar file %s\n", tarfile);

    retval = extract_tar(tarfile, rootfs_dir);

    return(retval);
}


