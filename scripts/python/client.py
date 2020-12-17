# this script adds a target to tcpprobe with 10 seconds
# time interval and gets the tcpprobe metrics then shows
# them on the console. once you close the script, the
# target will be removed from tcpprobe service.

# please note, you need to run tcpprobe grpc server
# (tcpprobe -grpc) before run this script.
# tcpprobe protobuf files are available here:
# https://github.com/mehrdadrad/tcpprobe/tree/main/scripts/python
# more information: https://github.com/mehrdadrad/tcpprobe/wiki
# if you have questions or concerns please open an issue:
# https://github.com/mehrdadrad/tcpprobe/issues

import signal
import sys
import grpc
import tcpprobe_pb2
import tcpprobe_pb2_grpc

from google.protobuf import json_format


def tcpprobe():
    addr = "https://www.google.com"
    with grpc.insecure_channel('127.0.0.1:8082') as channel:
        stub = tcpprobe_pb2_grpc.TCPProbeStub(channel)
        stub.Add(tcpprobe_pb2.Target(
            addr=addr, interval="10s"))

        stream = stub.Get(tcpprobe_pb2.Target(
            addr=addr))

        def cancel_request(unused_signum, unused_frame):
            stub.Delete(tcpprobe_pb2.Target(addr=addr))
            stream.cancel()
            sys.exit(0)

        signal.signal(signal.SIGINT, cancel_request)
        for m in stream:
            print(json_format.MessageToDict(
                m.metrics))


if __name__ == '__main__':
    tcpprobe()
