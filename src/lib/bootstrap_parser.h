#ifndef __SINGULARITY_BOOTDEF_H_
#define __SINGULARITY_BOOTDEF_H_

    int singularity_bootdef_open(char *bootdef_path);
    void singularity_bootdef_rewind();
    void singularity_bootdef_close();

    char *singularity_bootdef_get_value(char *key);
    int singularity_bootdef_get_version();
    char *singularity_bootdef_section_find(char *section_name);
    char *singularity_bootdef_section_get(char **script, char *section_name);

#endif
