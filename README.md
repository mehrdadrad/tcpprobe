![tcpprobe](/docs/imgs/tp_logo.png) 


[![Github Actions](https://github.com/mehrdadrad/tcpprobe/workflows/build/badge.svg)](https://github.com/mehrdadrad/tcpprobe/actions?query=workflow%3Abuild) [![Go report](https://goreportcard.com/badge/github.com/mehrdadrad/tcpprobe)](https://goreportcard.com/report/github.com/mehrdadrad/tcpprobe)  [![Coverage Status](https://coveralls.io/repos/github/mehrdadrad/tcpprobe/badge.svg?branch=main)](https://coveralls.io/github/mehrdadrad/tcpprobe?branch=main)

**TCPProbe** is a tool and service for network path and service monitoring. It exposes information about socket’s underlying TCP session, TLS and HTTP (more than 60 metrics). you can run it through command line or as a service. the request is highly customizable and you can integrate it with your application through gRPC. it runs in a Kubernetes cluster as cloud native application and by adding annotations on pods allow a fine control of the probing process.

![tcpprobe](/docs/imgs/tcpprobe.png)

## Features
- TCP socket statistics
- TCP/IP request customization
- Prometheus exporter
- Probing multiple hosts
- Runs as service
- Kubernetes native
- gRPC interface

#### Documentation
* [Command's options](https://github.com/mehrdadrad/tcpprobe/wiki/command's-options)
* [Metrics](https://github.com/mehrdadrad/tcpprobe/wiki/metrics)
* [Helm Chart](https://github.com/mehrdadrad/tcpprobe/wiki/helm)
* [CLI tutorial](https://github.com/mehrdadrad/tcpprobe/wiki/command-line-tutorial)
* [gRPC](https://github.com/mehrdadrad/tcpprobe/wiki/grpc)

#### Command line ([download Linux binary](https://github.com/mehrdadrad/tcpprobe/releases/latest/download/tcpprobe)) 
```
tcpprobe -json https://www.google.com
```
```json
{"Target":"https://www.google.com","IP":"142.250.72.196","Timestamp":1607567390,"Seq":0,"State":1,"CaState":0,"Retransmits":0,"Probes":0,"Backoff":0,"Options":7,"Rto":204000,"Ato":40000,"SndMss":1418,"RcvMss":1418,"Unacked":0,"Sacked":0,"Lost":0,"Retrans":0,"Fackets":0,"LastDataSent":56,"LastAckSent":0,"LastDataRecv":0,"LastAckRecv":0,"Pmtu":9001,"RcvSsthresh":56587,"Rtt":1365,"Rttvar":446,"SndSsthresh":2147483647,"SndCwnd":10,"Advmss":8949,"Reordering":3,"RcvRtt":0,"RcvSpace":62727,"TotalRetrans":0,"PacingRate":20765147,"BytesAcked":448,"BytesReceived":10332,"SegsOut":10,"SegsIn":11,"NotsentBytes":0,"MinRtt":1305,"DataSegsIn":8,"DataSegsOut":3,"DeliveryRate":1785894,"BusyTime":4000,"RwndLimited":0,"SndbufLimited":0,"Delivered":4,"DeliveredCe":0,"BytesSent":447,"BytesRetrans":0,"DsackDups":0,"ReordSeen":0,"RcvOoopack":0,"SndWnd":66816,"TCPCongesAlg":"cubic","HTTPStatusCode":200,"HTTPRcvdBytes":14683,"HTTPRequest":113038,"HTTPResponse":293,"DNSResolve":2318,"TCPConnect":1421,"TLSHandshake":57036,"TCPConnectError":0,"DNSResolveError":0}
```

#### Docker
```
docker run --rm mehrdadrad/tcpprobe 54.153.75.189:22
```

#### Docker Compose
TCPProbe and Prometheus
```
docker-compose up -d
```
Open your browser and try http://localhost:9090
You can edit the docker-compose.yml to customize the options and target(s).

#### Helm Chart
Detailed installation instructions for TCPProbe on Kubernetes are found [here](https://github.com/mehrdadrad/tcpprobe/wiki/helm).
```
helm install tcpprobe tcpprobe
```

## License
This project is licensed under MIT license. Please read the LICENSE file.

## Contribute
Welcomes any kind of contribution, please follow the next steps:

- Fork the project on github.com.
- Create a new branch.
- Commit changes to the new branch.
- Send a pull request.
