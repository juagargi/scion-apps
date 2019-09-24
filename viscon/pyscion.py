from ctypes import *
from collections import namedtuple
from ipaddress import IPv4Address

lib = cdll.LoadLibrary('./scion_api.so')


class SCIONException(Exception):
    """ A generic error from SCION """
    def __init__(self, error_msg):
        if isinstance(error_msg, bytes):
            error_msg = cstr_to_str(error_msg)
        super().__init__(error_msg)

charptr      = POINTER(c_char)
constcharptr = c_char_p

def str_to_cstr(str):
    return str.encode('utf-8')
def cstr_to_str(cstr):
    return cast(cstr, c_char_p).value.decode('utf-8')

 # ---------------------------------------------------------

class _PathInterface(Structure):
    _fields_ = [('isdAs', c_char_p), 
                ('ifid', c_size_t)]

class _FwdPathMeta(Structure):
    _fields_ = [('fwdPath_length', c_size_t),
                ('fwdPath', POINTER(c_ubyte)),
                ('mtu', c_ushort),
                ('interfaces_length', c_size_t),
                ('interfaces', POINTER(_PathInterface)),
                ('expTime', c_uint)]

class _HostInfo(Structure):
    _fields_ =[('port', c_ushort),
               ('ipv4', c_ubyte *4)]

class _PathReplyEntry(Structure):
    _fields_ = [('path', POINTER(_FwdPathMeta)),
                ('hostInfo', _HostInfo)]


 # ---------------------------------------------------------

HostInfo = namedtuple('HostInfo', ['ipv4', 'port'])
FwdPathMeta = namedtuple('FwdPathMeta', ['mtu', 'exp_time', 'fwd_path', 'interfaces'])
Interface = namedtuple('Interface', ['isd_as', 'if_id'])


class Path:
    def __init__(self, cpath):
        ip = IPv4Address(bytes(cpath.hostInfo.ipv4))
        self.host_info = HostInfo(
            ipv4=IPv4Address(bytes(cpath.hostInfo.ipv4)),
            port=cpath.hostInfo.port
        )
        p = cpath.path.contents
        interfaces = list(
            Interface(
                isd_as=cstr_to_str(p.interfaces[i].isdAs),
                if_id=int(p.interfaces[i].ifid) )
            for i in range(p.interfaces_length)
        )
        self.path = FwdPathMeta(
            mtu=p.mtu,
            exp_time=p.expTime,
            fwd_path=bytes( (p.fwdPath[i] for i in range(p.fwdPath_length)) ),
            interfaces=interfaces,
        )

    def __str__(self):
        path = self.path.interfaces
        return '{thisclass}({hostinfo}, {path})'.format(
            thisclass=type(self).__name__,
            hostinfo=str(self.host_info),
            path=path)

    def to_cstruct(self):
        centry = _PathReplyEntry()
        centry.hostInfo.port = self.host_info.port
        centry.hostInfo.ipv4 = (c_ubyte*4).from_buffer_copy(self.host_info.ipv4.packed)

        path = _FwdPathMeta()
        path.mtu = self.path.mtu
        path.expTime = self.path.exp_time
        path.fwdPath_length = len(self.path.fwd_path)
        path.fwdPath = POINTER(c_ubyte)()
        path.fwdPath = cast(self.path.fwd_path, POINTER(c_ubyte))
        path.interfaces_length = len(self.path.interfaces)
        interfaces = (_PathInterface * len(self.path.interfaces))()
        for i in range(len(self.path.interfaces)):
            interfaces[i].ifid = self.path.interfaces[i].if_id
            interfaces[i].isdAs = str_to_cstr(self.path.interfaces[i].isd_as)
        path.interfaces = POINTER(_PathInterface)()
        path.interfaces = interfaces

        centry.path = POINTER(_FwdPathMeta)()
        centry.path.contents = path
        return centry


class Paths:
    """ A path from the source AS to the destination, as returned by SCION """
    def __init__(self, paths, paths_len):
        self._paths = [Path(paths[i]) for i in range(paths_len.value)]

    def __str__(self):
        return '\n'.join( ('[{i}] {p}'.format(
            i=i,
            p=self._paths[i]) for i in range(len(self._paths))) )

    def __len__(self):
        return len(self._paths)

    def __getitem__(self, index):
        return self._paths[index]

    def __setitem__(self, index, value):
        self._paths[index] = value

    def __delitem__(self, index):
        del self._paths[index]

 # ---------------------------------------------------------


lib.Init.restype = c_char_p
def init():
    err = lib.Init()
    if err != None:
        raise SCIONException(err)


lib.LocalAddress.argtypes = [POINTER(charptr)]
lib.LocalAddress.restype = c_char_p
def local_address():
    ptr = charptr()
    err = lib.LocalAddress(byref(ptr))
    if err != None:
        raise SCIONException(err)
    return cstr_to_str(ptr)


lib.Paths.argtypes = [POINTER(c_size_t), POINTER(POINTER(_PathReplyEntry)), charptr]
lib.Paths.restype = c_char_p
lib.FreePathsMemory.argtypes = [POINTER(_PathReplyEntry), c_size_t]
lib.FreePathsMemory.restype = c_char_p
def paths(destination):
    paths_n = c_size_t()
    paths = (POINTER(_PathReplyEntry))()
    err = lib.Paths(byref(paths_n), byref(paths), str_to_cstr(destination))
    if err != None:
        raise SCIONException(err)
    pypaths = Paths(paths, paths_n)
    err = lib.FreePathsMemory(paths, paths_n)
    if err != None:
        raise SCIONException(err)
    return pypaths


lib.Open.argtypes = [POINTER(c_long), charptr, POINTER(_PathReplyEntry)]
lib.Open.restype = c_char_p
def open(destination, path):
    fd = c_long()
    cpath = path.to_cstruct()
    err = lib.Open(byref(fd), str_to_cstr(destination), cpath)
    if err != None:
        raise SCIONException(err)
    return int(fd.value)


lib.Close.argtypes = [c_long]
lib.Close.restype = c_char_p
def close(fd):
    err = lib.Close(fd)
    if err != None:
        raise SCIONException(err)


lib.Write.argtypes = [c_long, POINTER(c_char), c_size_t]
lib.Write.restype = c_char_p
def write(fd, buff):
    cbuff = (c_char * len(buff))(*buff)
    err = lib.Write(fd, cbuff, len(buff))
    if err != None:
        raise SCIONException(err)


lib.Listen.argtypes = [POINTER(c_long), c_ushort]
lib.Listen.restype = c_char_p
def listen(port):
    fd = c_long()
    err = lib.Listen(byref(fd), c_ushort(port))
    if err != None:
        raise SCIONException(err)
    return fd


lib.Read.argtypes = [POINTER(c_size_t), POINTER(charptr), c_long, POINTER(c_ubyte), c_size_t]
lib.Read.restype = c_char_p
def read(fd, buffer):
    c_buffer = (c_ubyte * len(buffer)).from_buffer(buffer)
    n = c_size_t()
    client_address = charptr()
    err = lib.Read(byref(n), byref(client_address), fd, c_buffer, len(buffer))
    if err != None:
        raise SCIONException(err)
    return cstr_to_str(client_address), int(n.value)

