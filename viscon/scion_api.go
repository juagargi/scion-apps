package main

/*
#include <stdlib.h>
typedef struct {
	char  *isdAs;
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
	"github.com/scionproto/scion/go/lib/addr"
	"github.com/scionproto/scion/go/lib/common"
	"github.com/scionproto/scion/go/lib/hostinfo"
	"github.com/scionproto/scion/go/lib/sciond"
	"github.com/scionproto/scion/go/lib/snet"
	"github.com/scionproto/scion/go/lib/spath"
)

var localAddress *snet.Addr
var sciondPath string
var connections = make(map[int]snet.Conn)

// -------------------------------------------------------------------------

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
	return nil
}

//export Paths
func Paths(retPathsLength *C.size_t, retPaths **C.PathReplyEntry, pDst *C.char) CError {
	dst := C.GoString(pDst)
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

	*retPathsLength = C.size_t(len(pathSet))
	*retPaths = (*C.PathReplyEntry)(C.malloc(C.size_t(len(pathSet) * C.sizeof_PathReplyEntry)))
	base := unsafe.Pointer(*retPaths)
	i := 0
	for _, p := range pathSet {
		centry := (*C.PathReplyEntry)(unsafe.Pointer(uintptr(base) + C.sizeof_PathReplyEntry*uintptr(i)))
		pathReplyEntryToCStruct(centry, p.Entry)
		i++
	}
	return nil
}

func pathReplyEntryToCStruct(centry *C.PathReplyEntry, srcEntry *sciond.PathReplyEntry) {
	path := fwdPathMetaToCStruct(srcEntry.Path)
	hostInfo := C.HostInfo{
		port: C.ushort(srcEntry.HostInfo.Port),
	}
	if len(srcEntry.HostInfo.Addrs.Ipv4) > 0 {
		for i := 0; i < 4; i++ {
			hostInfo.ipv4[i] = C.uchar(srcEntry.HostInfo.Addrs.Ipv4[i])
		}
	}
	centry.path = path
	centry.hostInfo = hostInfo
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

//export FreePathsMemory
func FreePathsMemory(paths *C.PathReplyEntry, paths_len C.size_t) CError {
	base := unsafe.Pointer(paths)
	for i := 0; i < int(paths_len); i++ {
		entry := (*C.PathReplyEntry)(unsafe.Pointer(uintptr(base) + C.sizeof_PathReplyEntry*uintptr(i)))
		C.free(unsafe.Pointer(entry.path.fwdPath))
		C.free(unsafe.Pointer(entry.path.interfaces))
	}
	C.free(base)
	return nil
}

func findFreeDescriptor(conn snet.Conn) (int, error) {
	k := 0
	for ; k < 4096; k++ {
		if _, found := connections[k]; !found {
			break
		}
	}
	if k == 4096 {
		return 0, fmt.Errorf("Connection descriptor table full")
	}
	connections[k] = conn
	return k, nil
}

//export Open
func Open(pFd *C.long, pHostAddress *C.char, cpath *C.PathReplyEntry) CError {
	dst := C.GoString(pHostAddress)
	dstAddress, err := snet.AddrFromString(dst)
	if err != nil {
		return errorToCString(err)
	}
	dbg("Go: opened %v ; host addr: %+v", dstAddress, dstAddress.Host)
	dbg("L3 = %+v ; L4 = %+v", dstAddress.Host.L3, dstAddress.Host.L4)
	if dstAddress.Host.L4 == nil {
		return cerr("Unspecified port number")
	}
	path, err := cPathToPathReplyEntry(cpath)
	if err != nil {
		return errorToCString(err)
	}
	if !dstAddress.IA.Equal(localAddress.IA) {
		dstAddress.Path = spath.New(path.Path.FwdPath)
		dstAddress.Path.InitOffsets()
		dstAddress.NextHop, _ = path.HostInfo.Overlay()
	}
	conn, err := snet.DialSCION("udp4", localAddress, dstAddress)
	if err != nil {
		return errorToCString(err)
	}
	fd, err := findFreeDescriptor(conn)
	if err != nil {
		return errorToCString(err)
	}
	*pFd = C.long(fd)
	return nil
}

func cPathToPathReplyEntry(centry *C.PathReplyEntry) (*sciond.PathReplyEntry, error) {
	interfaces := make([]sciond.PathInterface, centry.path.interfaces_length)
	for i := 0; i < len(interfaces); i++ {
		pi := (*C.PathInterface)(unsafe.Pointer(uintptr(unsafe.Pointer(centry.path.interfaces)) + C.sizeof_PathInterface*uintptr(i)))
		ia, err := addr.IAFromString(C.GoString(pi.isdAs))
		if err != nil {
			return nil, err
		}
		interfaces[i] = sciond.PathInterface{
			IfID:     common.IFIDType(pi.ifid),
			RawIsdas: ia.IAInt(),
		}

	}
	entry := &sciond.PathReplyEntry{
		Path: &sciond.FwdPathMeta{
			Mtu:        uint16(centry.path.mtu),
			ExpTime:    uint32(centry.path.expTime),
			FwdPath:    C.GoBytes(unsafe.Pointer(centry.path.fwdPath), C.int(centry.path.fwdPath_length)),
			Interfaces: interfaces,
		},
		HostInfo: hostinfo.HostInfo{
			Addrs: struct {
				Ipv4 []byte
				Ipv6 []byte
			}{
				Ipv4: C.GoBytes(unsafe.Pointer(&centry.hostInfo.ipv4), 4),
			},
			Port: uint16(centry.hostInfo.port),
		},
	}
	return entry, nil
}

//export Close
func Close(fd C.long) CError {
	conn, found := connections[int(fd)]
	if !found {
		return cerr("Bad descriptor")
	}
	err := conn.Close()
	delete(connections, int(fd))
	if err != nil {
		return errorToCString(err)
	}
	return nil
}

//export Write
func Write(fd C.long, bytes *C.uchar, count C.size_t) CError {
	conn, found := connections[int(fd)]
	if !found {
		return cerr("Bad descriptor")
	}
	n, err := conn.Write(C.GoBytes(unsafe.Pointer(bytes), C.int(count)))
	dbg("After write, err = %v", err)
	if err != nil {
		return errorToCString(err)
	}
	dbg("Written %d bytes", n)
	return nil
}

//export Listen
func Listen(pFd *C.long, port C.ushort) CError {
	if !initialized() {
		return cerr("Not initialized")
	}
	serverAddress := localAddress.Copy()
	serverAddress.Host.L4 = addr.NewL4UDPInfo(uint16(port))
	conn, err := snet.ListenSCION("udp4", serverAddress)
	if err != nil {
		return errorToCString(err)
	}
	fd, err := findFreeDescriptor(conn)
	if err != nil {
		return errorToCString(err)
	}
	*pFd = C.long(fd)
	return nil
}

//export Read
func Read(bytesRead *C.size_t, pClientAddr **C.char, fd C.long, buffer *C.uchar, bufLength C.size_t) CError {
	conn, found := connections[int(fd)]
	if !found {
		return cerr("Bad descriptor")
	}
	buff := make([]byte, int(bufLength))
	n, clientAddr, err := conn.ReadFromSCION(buff)
	if err != nil {
		return errorToCString(err)
	}
	dbg("Read %d bytes from %s", n, clientAddr)
	*bytesRead = C.size_t(n)
	*pClientAddr = C.CString(clientAddr.String())
	// copy bytes using an unsafe.Pointer "cast" (aka void*)
	tmpArray := (*[1 << 30]byte)(unsafe.Pointer(buffer))
	copy(tmpArray[:], buff)

	return nil
}

func main() {}
