from pyscion import *



def main():
    # add()
    # add()
    # add()
    
    init()
    print('Local Address is %s' % local_address())
    paths("17-ffaa:1:a,[127.0.0.1]")
    p = paths("20-ffaa:0:1401,[127.0.0.1]")
    print('Got %d paths' % len(p))

if __name__ == "__main__":
    main()


