from ctypes import *

lib = cdll.LoadLibrary('./scion_api.so')


class SCIONException(Exception):
    """ A generic error from SCION """

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


lib.Paths.argtypes = [POINTER(c_size_t), charptr]
lib.Paths.restype = c_char_p
def paths(destination):
    paths_n = c_size_t()
    err = lib.Paths(byref(paths_n), str_to_cstr(destination))
    if err != None:
        raise SCIONException(err)
    print('[DEBUG Paths] Got %d paths' % paths_n.value)


