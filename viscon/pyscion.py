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
    # return cstr.decode('utf-8')
    # return cstr.contents.value
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

 # ---------------------------------------------------------


lib.Add.argtypes = []
lib.restype = None
def add():
    lib.Add()
    # print("python add")


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


