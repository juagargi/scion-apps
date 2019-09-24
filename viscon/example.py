from pyscion import *



def main():
    # add()
    # add()
    # add()
    
    init()
    print('Local Address is %s' % local_address())
    p = paths("17-ffaa:0:1107,[127.0.0.1]")
    # p = paths("20-ffaa:0:1401,[127.0.0.1]:8080")
    print('Got %d paths' % len(p))
    # fd = open("20-ffaa:0:1401,[127.0.0.1]:8080", p[0])
    fd = open("17-ffaa:0:1107,[127.0.0.1]:12345", p[0])
    write(fd, b'abcd')
    close(fd)

if __name__ == "__main__":
    main()


