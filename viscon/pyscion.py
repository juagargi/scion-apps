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
    _fields_ = [('path', _FwdPathMeta),
                ('hostInfo', _HostInfo)]


 # ---------------------------------------------------------

namedtuple('HostInfo', ['port', 'ipv4'])
namedtuple('PathReplyEntry', ['path', 'host_info'])

class Paths():
    """ A path from the source AS to the destination, as returned by SCION """
    def __init__(self, paths, paths_len):


        p = paths.contents
        print(p.hostInfo)
        print(p.hostInfo.ipv4)
        print(p.hostInfo.ipv4[0])
        print(p.hostInfo.port)
        exit(0)

        self._paths = []
        for i in range(0, paths_len.value):
            p = paths[i]
            print(p)
            ip = IPv4Address(bytes([p.hostInfo.ipv4[0], p.hostInfo.ipv4[1], p.hostInfo.ipv4[2], p.hostInfo.ipv4[3]]) )
            print('IP = ', ip)
            ip = IPv4Address(bytes( [p.hostInfo.ipv4[i] for i in (0,1,2,3)] ))
            print('IP = ', ip)
            self._paths.append(p)

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


# lib.Paths.argtypes = [POINTER(c_size_t), POINTER(POINTER(_PathReplyEntry)), charptr]
lib.Paths.argtypes = [POINTER(POINTER(_HostInfo)), POINTER(c_size_t), POINTER(POINTER(_PathReplyEntry)), charptr]
lib.Paths.restype = c_char_p
def paths(destination):
    paths_n = c_size_t()
    paths = (POINTER(_PathReplyEntry))()
    # paths = pointer((POINTER(_PathReplyEntry))())
    deleteme = (POINTER(_HostInfo))()
    
    err = lib.Paths(byref(deleteme), byref(paths_n), byref(paths), str_to_cstr(destination))
    # err = lib.Paths(byref(paths_n), byref(paths), str_to_cstr(destination))
    # # err = lib.Paths(byref(paths_n), paths, str_to_cstr(destination))
    if err != None:
        raise SCIONException(err)
    
    print('Py Final p = %s, *p = %s' % (byref(deleteme), deleteme.contents))
    print('Hostinfo.port = %s' % deleteme[1].port)
    # exit(0)
    print('[DEBUG Paths] Got %d paths' % paths_n.value)

    print('[DEBUG Paths] Got paths p = %s , *p = %s' % (paths, paths.contents))
    print('[DEBUG Paths] hostinfo[0].port = %s' % paths[0].hostInfo.port)
    exit(0)
    # # print(dir(paths))
    # print(paths.contents)
    # for i in range(0, paths_n.value):
    #     print(i)
    # # print('[DEBUG Paths] mtu = %d' % meta.mtu)
    # # print('[DEBUG Paths] one isd as: %s' % meta.interfaces[0].isdAs)
    Paths(paths, paths_n)


