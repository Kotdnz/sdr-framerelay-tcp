
# sdr-framerelay-tcp
The common idea is to compress sdr data and don't touch cmd channel.
Thus, we have two different configuration at the transport ends - encode from remote
 end and decode on the local:

remote end[sdr_tcp:9001 <-> localhost:9002] <--- internet ---> client end [framerelay connect to remoteIP:9002: <-> client conneted to localhost:9001]

How to build<br>
<code>go get .
go build 
./sdr-framerelay-tcp.go -h
</code>