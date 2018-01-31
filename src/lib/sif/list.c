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

#include <stdlib.h>

#include "list.h"

Node *
listcreate(void *elem)
{
	Node *new = malloc(sizeof(Node));
	if(new == NULL)
		return NULL;
	new->elem = elem;
	new->next = NULL;
	return new;
}

void
listaddfront(Node *head, Node *new)
{
	new->next = head->next;
	head->next = new;
}

void
listaddtail(Node *head, Node *new)
{
	Node *p;

	if(head->next == NULL){
		head->next = new;
		return;
	}
	for(p = head->next; p != NULL; p = p->next){
		if(p->next == NULL){
			p->next = new;
			break;
		}
	}
}

Node *
listfind(Node *head, void *elem, Searchfn fn)
{
	Node *e;

	for(e = head->next; e != NULL; e = e->next){
		if(fn){
			if(fn(e->elem, elem))
				return e;
		}else{
			if(e->elem == elem)
				return e;
		}
	}
	return NULL;
}

Node *
listdelete(Node *head, void *elem, Searchfn fn)
{
	Node *prev = head;
	Node *e;

	for(e = head->next; e != NULL; prev = e, e = e->next){
		if(fn){
			if(fn(e->elem, elem)){
				prev->next = e->next;
				return e;
			}
		}else{
			if(e->elem == elem){
				prev->next = e->next;
				return e;
			}
		}
	}
	return NULL;
}

int
listforall(Node *head, Actionfn fn, void *data)
{
	int ret;
	Node *e;

	for(e = head->next; e != NULL; e = e->next){
		ret = fn(e->elem, data);
		if(ret < 0)
			return ret;
	}

	return 0;
}
