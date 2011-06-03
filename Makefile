include $(GOROOT)/src/Make.inc

TARG=github.com/nicolaspaton/goapn
GOFILES=\
	apn.go\
	notification.go\
    queue.go\
    service.go\

include $(GOROOT)/src/Make.pkg

