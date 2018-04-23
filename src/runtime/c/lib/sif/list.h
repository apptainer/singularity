/*
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * Copyright (c) 2017, Yannick Cote <yanick@divyan.org>. All rights reserved.
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

typedef struct Node Node;
struct Node{
	void *elem;
	Node *next;
};

typedef int (*Searchfn)(void *cur, void *elem);
typedef int (*Actionfn)(void *elem, void *data);

Node *listcreate(void *elem);
void listaddfront(Node *head, Node *new);
void listaddtail(Node *head, Node *new);
Node *listfind(Node *head, void *elem, Searchfn fn);
Node *listdelete(Node *head, void *elem, Searchfn fn);
int listforall(Node *head, Actionfn fn, void *data);
