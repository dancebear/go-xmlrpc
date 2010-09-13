# Copyright 2009 The Go Authors.  All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

include $(GOROOT)/src/Make.inc

TARG=rpc2/xmlrpc

GOFILES=\
	marshal.go\
	unmarshal.go\
	xmlrpccodec.go\
	client.go\
	server.go\

include $(GOROOT)/src/Make.pkg

GC+=-I../_obj
LD+=-L../_obj
