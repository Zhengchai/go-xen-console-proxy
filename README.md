# Console proxy for Xenserver

This is a go implementation of a Xenserver console proxy to be used
in Cloudstack. This enables the use of modern clients like noVNC
which have better support for scrollback and copy.

# Build

To build this console proxy, just run the `Makefile`. This will compile all the
code along with the static files that are required to display the interface


```
cd /path/to/repo
make
```

# Install

Copy the generated binary to the Cloudstack repo and run the systemvm build to generate a new `systemvm.iso`


```
cp go-xen-console-proxy ./systemvm/patches/debian/config/opt/cloud/bin/go-xen-console-proxy
```

replace the new `systemvm.iso` to the Xen host and destroy the console VM from cloudstack. A new console proxy VM will be 
created which will have the proxy.


# High level workflow

1. Browser calls the management server with a URL to get the console with `websocketconsole=true` added to the query params

```
http://172.16.21.148:8080/client/console?cmd=access&vm=b4ee0960-d60a-4013-a24f-6e30733f0a2e&websocketconsole=true
```

2. The management server verifies the user keys/access and returns a URL which points to the public IP of the console proxy VM. 
The URL also contains a `token` which is an encrypted string containing information about the VNC session. Note that the new console
proxy is running on port `9090` (the original console proxy is still running on port 80)

```
http://172.31.2.190:9090/console?token=cDiJpVXbkMSG_GiyISA5WIfiy8UzKzRKV73b4UIpnneIbexXtMzwqKUkQ9NPxh6zivm6Eja29EuQCBq-3I6_oQ0IOpQK3amD5xo6BgBZAM0OTow0zd3e9R5AqQyhqoHYTR0bUe-lxap6bTXrEMY01IKmqc7Kkbqo6tUUdU9Y9-X7HBQfJcvZxA5pX-WQ5c8KRdN5cBfekU-os12vJFbk9lV36DqUQioF2bo5xKu4YHJ0AMUjcavQw3uDUbOpE2Ily1mRm5f7h9HnFyFvVy9Ob5EBOpSxz2KD796r77-dxEofr6f4bBtf_LncKAy9GhaGXrZpWp6UZA0b75_PpUYKXnqZCpXx5Q6-i37kayzeXW-FNnQDCzbNydg-32mbDls2fD14s6a11jgVHrBWpgCAV1z0CX8TWILaBYAm2Z3KRgjKOYeoSs6kwdVASzqvH-RU8-hLem7P_d5u8bB4kdR385k2st-YDMTKZ_ON07JO6KQ
```

3. The client calls the console proxy's public IP which sets up a backend VNC session to xenserver and redirects to the noVNC UI along with a session.

```
http://172.31.2.190:9090/vnc.html?path=d965e329-c32b-2c9c-a33c-66cafe6214c3
```

4. The client does a webscoket call to the endpoint 

```
http://172.31.2.190:9090/vnc/<session UUID>
```

which sets up the proxy which starts forwading traffic back and forth between the browser and the Xenserver
