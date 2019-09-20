package main

/*
#include <stdlib.h>
typedef struct {
	char *isdAs;
	long   ifid;
} PathInterface;

typedef struct {
	size_t         fwdPath_length;
	unsigned char *fwdPath;

	unsigned short mtu;

	size_t         interfaces_length;
	PathInterface *interfaces;

	unsigned int expTime;
} FwdPathMeta;

typedef struct {
	unsigned short port;
	unsigned char  ipv4[4];
} HostInfo;

typedef struct {
	FwdPathMeta *path;
	HostInfo    hostInfo;
} PathReplyEntry;
*/
import "C"

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/netsec-ethz/scion-apps/lib/scionutil"
	"github.com/scionproto/scion/go/lib/sciond"
	"github.com/scionproto/scion/go/lib/snet"
)

var localAddress *snet.Addr
var sciondPath string

var deleteme = 0

//export Add
func Add() {
	deleteme = deleteme + 1
	dbg("Go Add, deleteme = %d", deleteme)
}

func dbg(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

type CError *C.char

func cerr(err string) CError {
	return C.CString(err)
}
func errorToCString(err error) CError {
	if err == nil {
		return nil
	}
	return C.CString(err.Error())
}

func initialized() bool {
	return localAddress != nil
}

// -------------------------------------------------------------------------

//export Init
func Init() CError {
	if initialized() {
		return cerr("Already initialized")
	}
	// find local IA, dispatcher and sciond paths
	localAddressStr, err := scionutil.GetLocalhostString()
	if err != nil {
		errorToCString(err)
	}
	localAddress, err = snet.AddrFromString(localAddressStr)
	if err != nil {
		errorToCString(err)
	}
	dispatcherPath := scionutil.GetDefaultDispatcher()
	sciondPath := sciond.GetDefaultSCIONDPath(nil)
	err = snet.Init(localAddress.IA, sciondPath, dispatcherPath)
	if err != nil {
		errorToCString(err)
	}
	return nil
}

//export LocalAddress
func LocalAddress(retLocalAddress **C.char) CError {
	if !initialized() {
		return cerr("Not initialized")
	}
	if localAddress == nil {
		return cerr("Exception: local address is nil")
	}
	cstr := C.CString(localAddress.String())
	*retLocalAddress = cstr
	_ = cstr
	return nil
}

//export Paths
func Paths(paths_length *C.size_t, pDst *C.char) CError {
	dst := C.GoString(pDst)
	dbg("Go method Paths called with (%v) = %s", pDst, dst)
	if !initialized() {
		return cerr("Not initialized")
	}
	dstAddress, err := snet.AddrFromString(dst)
	if err != nil {
		return errorToCString(err)
	}
	pathMgr := snet.DefNetwork.PathResolver()
	pathSet := pathMgr.Query(context.Background(), localAddress.IA, dstAddress.IA, sciond.PathReqFlags{})
	if len(pathSet) == 0 {
		return cerr("No paths")
	}

	*paths_length = C.size_t(len(pathSet))
	for _, p := range pathSet {
		dbg("%d %s", len(p.Entry.Path.Interfaces)/2, p.Entry.Path.String())
		var path *C.PathReplyEntry
		path = pathReplyEntryToCStruct(p.Entry)
		_ = path
	}
	return nil
}

func pathReplyEntryToCStruct(entry *sciond.PathReplyEntry) *C.PathReplyEntry {
	path := fwdPathMetaToCStruct(entry.Path)
	hostInfo := C.HostInfo{
		port: C.ushort(entry.HostInfo.Port),
	}
	for i := 0; i < 4; i++ {
		hostInfo.ipv4[i] = C.uchar(entry.HostInfo.Addrs.Ipv4[i])
	}
	return &C.PathReplyEntry{
		path:     path,
		hostInfo: hostInfo,
	}
	_ = path
	return nil
}

func fwdPathMetaToCStruct(src *sciond.FwdPathMeta) *C.FwdPathMeta {
	ret := C.FwdPathMeta{}
	ret.mtu = C.ushort(src.Mtu)
	ret.expTime = C.uint(src.ExpTime)
	ret.fwdPath_length = C.size_t(len(src.FwdPath))
	ret.fwdPath = (*C.uchar)(C.malloc(C.size_t(ret.fwdPath_length)))
	// a byte is a byte in Go and C. Cast the C array of bytes to a Go array of bytes by using
	// an unsafe.Pointer (aka void*)
	tmpArray := (*[1 << 30]byte)(unsafe.Pointer(ret.fwdPath))
	copy(tmpArray[:], src.FwdPath)

	ret.interfaces_length = C.size_t(len(src.Interfaces))
	ret.interfaces = (*C.PathInterface)(C.malloc(C.size_t(ret.interfaces_length * C.sizeof_PathInterface)))
	base := unsafe.Pointer(ret.interfaces)
	for i := 0; i < len(src.Interfaces); i++ {
		p := unsafe.Pointer(uintptr(base) + C.sizeof_PathInterface*uintptr(i))
		cpi := (*C.PathInterface)(p)
		pathInterfaceToCStruct(cpi, &src.Interfaces[i])
	}
	return &ret
}

func pathInterfaceToCStruct(cpi *C.PathInterface, pi *sciond.PathInterface) {
	cpi.isdAs = C.CString(pi.RawIsdas.String())
	cpi.ifid = C.long(pi.IfID)
}

func main() {}
