
# sdr-framerelay-tcp
The common idea is to compress sdr data with 8Mb buffer and don't touch cmd channel.
Thus, we have two different configuration at the transport ends
<br>
<img src="https://github.com/Kotdnz/sdr-framerelay-tcp/blob/main/sdr_v1-option%202.drawio.png"/>

How to build
[code] go get . 
go build sdr-framerelay-tcp.go
[/code]
