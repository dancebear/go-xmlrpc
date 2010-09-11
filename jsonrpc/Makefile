# Copyright 2009 The Go Authors.  All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

include $(GOROOT)/src/Make.inc

#package: jsonrpc

TARG=rpc2/jsonrpc

GOFILES=\
	jsonrpccodec.go\

include $(GOROOT)/src/Make.pkg

GC+=-I../_obj
LD+=-L../_obj -lrpc2
