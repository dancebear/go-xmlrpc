# Copyright 2009 The Go Authors.  All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

include $(GOROOT)/src/Make.$(GOARCH)

daqstatus: daqstatus.6
	6l -o daqstatus -L _obj daqstatus.6

daqstatus.6: daqstatus.go _obj/xmlrpc.a
	6g -I _obj daqstatus.go

clean_ds:
	${RM} daqstatus

clean: clean_ds
package: xmlrpc

TARG=xmlrpc

GOFILES=\
	xmlrpc.go\
	xmlrpccodec.go\
	jsonrpccodec.go\

include $(GOROOT)/src/Make.pkg
