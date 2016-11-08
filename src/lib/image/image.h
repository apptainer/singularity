#ifndef __SINGULARITY_IMAGE_H_
#define __SINGULARITY_IMAGE_H_

extern int singularity_image_extern_create(int argc, char ** argv);
extern int singularity_image_extern_expand(int argc, char ** argv);
extern int singularity_image_mount(int argc, char ** argv);
extern int singularity_image_bind(int argc, char ** argv);

extern int singularity_image_check(FILE *image_fp);
extern int singularity_image_offset(FILE *image_fp);
extern int singularity_image_create(char *image, int size);
extern int singularity_image_expand(char *image, int size);

#endif
