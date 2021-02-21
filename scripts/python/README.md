### Sample python client

#### Requirements
```
#python3 -m pip install grpcio protobuf
```

First make sure the service is up and running
```
tcpprobe -grpc -q
```
Run the sample client
```
python3 client.py
```

```json
{'HTTPRequest': 73152.0, 'DataSegsIn': 7.0, 'Sacked': 0.0, 'TotalRetrans': 0.0, 'BytesRetrans': 0.0, 'State': 1.0, 'RcvRtt': 0.0, 'DataSegsOut': 3.0, 'LastDataSent': 56.0, 'Rttvar': 433.0, 'Pmtu': 9001.0, 'BytesReceived': 9418.0, 'HTTPStatusCode': 200.0, 'Ato': 40000.0, 'Rtt': 1307.0, 'BytesSent': 447.0, 'CaState': 0.0, 'DNSResolveError': 0.0, 'TLSHandshake': 18343.0, 'Retrans': 0.0, 'SndWnd': 66816.0, 'MinRtt': 1239.0, 'LastAckSent': 0.0, 'LastAckRecv': 0.0, 'TCPConnectError': 0.0, 'NotsentBytes': 0.0, 'Unacked': 0.0, 'Lost': 0.0, 'BytesAcked': 448.0, 'PacingRate': 21686102.0, 'RcvOoopack': 0.0, 'SndbufLimited': 0.0, 'Options': 7.0, 'Rto': 204000.0, 'SegsOut': 9.0, 'DeliveredCe': 0.0, 'DNSResolve': 669.0, 'SndCwnd': 10.0, 'Fackets': 0.0, 'SegsIn': 10.0, 'RcvSpace': 62727.0, 'Reordering': 3.0, 'TCPConnect': 1359.0, 'RcvSsthresh': 56587.0, 'SndSsthresh': 2147483647.0, 'Backoff': 0.0, 'RwndLimited': 0.0, 'RcvMss': 1418.0, 'Probes': 0.0, 'HTTPResponse': 279.0, 'SndMss': 1418.0, 'HTTPRcvdBytes': 12816.0, 'DsackDups': 0.0, 'BusyTime': 4000.0, 'Delivered': 4.0, 'TCPCongesAlg': 'cubic', 'ReordSeen': 0.0, 'Advmss': 8949.0, 'Retransmits': 0.0, 'LastDataRecv': 0.0, 'DeliveryRate': 1845152.0}
```
