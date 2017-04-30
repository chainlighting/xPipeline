#### xPipeline
----
xPipeline can build a data process pipeline by links of **Consumers**|**Streams**|**Producers**.

##### ==Consumer==
> the special consumer will collect data from indicated datasource,split messages from datasource stream.when the message be extraced,it will distribute message to ==Producer=='s channel based on ==Stream=='s setting.Consumer's collect transport can be:
* [x] TCP/UDP/TLS/HTTP Client Mode,as a client to connect/req datasource
* [x] TCP/UDP/TSL/HTTP Server Mode,as a server wait datasouce to connect/req
* [x] Files(local),collect data from local files
* [ ] Files(remote),collect data from remote files by FTP/SFTP/SCP
* [ ] Kafaka
* [ ] Mysql/Posgrel...
* [ ] ...

##### ==Stream==
> it's a bridge between ==Consumer== and ==Producer==,the Message's *transform*/*filter*/*distribute* policy will be bound to it.Stream's Policy can be:
* [x] Transform,...
* [x] Filter,...
* [x] Distribute,the distribute method can be BroadCast/Random/RoundRobin/Route.

##### ==Producer==
> producer get message from the special-streams,and process it.at the end of producer's progress,it can send the processed message to the next Pipeline.the sink transport can be:
* [x] TCP/UDP/TLS/HTTP Client Mode,as a client to connect/req datasource
* [x] TCP/UDP/TSL/HTTP Server Mode,as a server wait datasouce to connect/req
* [x] Files(local),collect data from local files
* [ ] Files(remote),collect data from remote files by FTP/SFTP/SCP
* [ ] Kafaka
* [ ] Mysql/Posgrel...
* [ ] ...