#!/usr/bin/python3

import json
import os
import sys
import subprocess

from time import sleep
from signal import signal,SIGINT

polling_wait = 2 # wait between checks, in seconds

name = os.path.basename(sys.argv[0])
ifaces = sys.argv[1:]

def usage():
    print(f"usage: {name} <iface1> [<iface2> ...]")

def sigint(signum, frame):
    print("caught SIGINT, exiting")
    exit(0)

def check_if_state():
    ready = True
    try:
        proc = subprocess.run([ '/sbin/ip', '-j', 'addr' ], check=True, stdin=subprocess.PIPE, stdout=subprocess.PIPE)
    except subprocess.CalledProcessError as e:
        print(f'got non-zero return code executing ip command {proc.returncode}, {str(e)}')
        exit(1)
    except FileNotFoundError as e:
        print(f'could not find the ip command, {str(e)}')
    
    ip = json.loads(proc.stdout)

    id = dict()
    for i in ip: 
        id[i['ifname']] = i

    violations = []
    for i in ifaces:
        if i not in id.keys():
            ready = False
            violations.append(f"{i} is not available")
            continue
        if id[i]['operstate'] != "UP":
            ready = False
            violations.append(f"{i} is present but link is not up")
            continue
        found = False
        for a in id[i]['addr_info']:
            if a["family"] == "inet":
                found = True
                break
        if not found:
            ready = False
            violations.append(f"{i} is does not have an ipv4 address")
    
    return ready, violations
        

def main():
    if len(ifaces) == 0:
        usage()
        exit(1)

    signal(SIGINT, sigint)
    print(f"waiting for ifaces [{', '.join(ifaces)}]")
    ready, violations = check_if_state()
    while not ready:
        print(f"ifaces [{', '.join(ifaces)}] are not ready, sleeping {polling_wait}s: [{'; '.join(violations)}]")
        sleep(polling_wait)
        ready, violations = check_if_state()
    print(f"ifaces [{', '.join(ifaces)}] are ready, exiting")


if __name__ == "__main__":
    main()