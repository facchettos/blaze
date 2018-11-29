# Blaze

Blaze is a file transfer application. It implements a new file transfer protocol which uses both TCP and UDP. Blaze was designed to increase throughput of file transfer when sender and receiver are linked by a high throughput link but with a high latency. For this kind of transfer, classic tcp based applications have a highly reduced throughput. By using tcp and udp we can have a reliable transfer protocol which is less impacted by the latency. 
