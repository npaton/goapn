include $(GOROOT)/src/Make.inc

TARG=apn
GOFILES=\
	apn.go\
	notification.go\
    queue.go\
    service.go\

include $(GOROOT)/src/Make.pkg

