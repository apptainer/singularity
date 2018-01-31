/*
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * Copyright (c) 2017, Yannick Cote <yhcote@gmail.com> All rights reserved.
 *
 * See the COPYRIGHT.md file at the top-level directory of this distribution and at
 * https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
 *
 * This file is part of the Singularity Linux container project. It is subject to the license
 * terms in the LICENSE.md file found in the top-level directory of this distribution and
 * at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
 * of Singularity, including this file, may be copied, modified, propagated, or distributed
 * except according to the terms contained in the LICENSE.md file.
 */

#ifndef __SINGULARITY_SIFACCESS_H_
#define __SINGULARITY_SIFACCESS_H_

char *sif_archstr(char *arch);
char *sif_hashstr(Sifhashtype htype);
char *sif_partstr(Sifparttype ptype);
char *sif_datastr(Sifdatatype dtype);
char *sif_fsstr(Siffstype ftype);

int sif_printrow(void *elem, void *data);
int sif_printdesc(void *elem, void *data);
void sif_printlist(Sifinfo *info);
void sif_printheader(Sifinfo *info);

Sifdescriptor *sif_getdescid(Sifinfo *info, int id);
Sifdescriptor *sif_getlinkeddesc(Sifinfo *info, int id);

Sifheader *sif_getheader(Sifinfo *info);
Sifdeffile *sif_getdeffile(Sifinfo *info, int groupid);
Siflabels *sif_getlabels(Sifinfo *info, int groupid);
Sifenvvar *sif_getenvvar(Sifinfo *info, int groupid);
Sifpartition *sif_getpartition(Sifinfo *info, int groupid);
Sifsignature *sif_getsignature(Sifinfo *info, int groupid);

#endif /* __SINGULARITY_SIFACCESS_H_ */
