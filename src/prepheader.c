/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 */

#include <sys/mman.h>
#include <sys/types.h>
#include <sys/stat.h>

#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include "lib/image/image.h"


/* prepend string "str" into file "filename" */
int
prepend(char *str, char *filename)
{
	int fd;
	char *map;
	struct stat st;

	if(filename == NULL){
		fprintf(stderr, "Error invalid filename\n");
		return -1;
	}
	fd = open(filename, O_RDWR);
	if(fd < 0){
		perror("Error while opening file");
		return -1;
	}
	if(lseek(fd, 0, SEEK_END) < 0){
		perror("Error while seeking to end of file");
		close(fd);
		return -1;
	}
	if(write(fd, str, strlen(str)) != (ssize_t)strlen(str)){
		perror("Error writing past end of file");
		close(fd);
		return -1;
	}
	if(fstat(fd, &st) < 0){
		perror("Error calling fstat()");
		close(fd);
		return -1;
	}
	map = mmap(NULL, st.st_size, PROT_WRITE, MAP_SHARED, fd, 0);
	if(map == MAP_FAILED){
		perror("Error mapping file");
		close(fd);
		return -1;
	}

	/* the heart of the program is these two lines really */
	memmove(map+strlen(str), map, st.st_size-strlen(str));
	strncpy(map, str, strlen(str));

	if(munmap(map, st.st_size) < 0){
		perror("Error tearing down map, file corrupted -- dont use");
		close(fd);
		return -1;
	}

	if(fchmod(fd, st.st_mode | S_IXUSR | S_IXGRP | S_IXOTH) < 0){
		perror("Error trying to change mode +x\n");
		close(fd);
		return 0;
	}

	if(close(fd) < 0){
		perror("Error closing file -- weird");
		return -1;
	}

	return 0;
}

int
main(int argc, char *argv[])
{
	if(argc != 2){
		fprintf(stderr, "usage: %s PART_IMAGE_FILE\n", argv[0]);
		return -1;
	}

	prepend(LAUNCH_STRING, argv[1]);

	return 0;
}
