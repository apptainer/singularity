#ifndef __SINGULARITY_IMAGE_H_
#define __SINGULARITY_IMAGE_H_

extern int singularity_image_extern_create(int argc, char ** argv);
extern int singularity_image_extern_expand(int argc, char ** argv);
extern int singularity_image_mount(int argc, char ** argv);
extern int singularity_image_bind(int argc, char ** argv);

#endif
