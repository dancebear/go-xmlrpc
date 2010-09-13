include $(GOROOT)/src/Make.inc

TARG=daqstatus
GOFILES=\
	daqstatus.go\

include $(GOROOT)/src/Make.cmd

GC+=-I ../_obj -I ../jsonrpc/_obj -I ../xmlrpc/_obj
LD+=-L ../_obj -L ../jsonrpc/_obj -L ../xmlrpc/_obj

