
# sdr-framerelay-tcp
The common idea is to compress sdr data with 8Mb buffer and don't touch cmd channel.
Thus, we have two different configuration at the transport ends

How to build<p>
<code>
go get .<br> 
go build sdr-framerelay-tcp.go
./sdr-framerelay-tcp.go -h
</code>
