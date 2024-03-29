# Arista Setup

Arista vEos Download
https://www.arista.com/en/support/software-download

Arista Go eAPI Library
https://github.com/aristanetworks/goeapi

# Setup VM

https://eos.arista.com/veos-and-virtualbox/

Configure VM
https://www.youtube.com/watch?v=nbDF7hzBPM0

Configure SSH
https://www.youtube.com/watch?v=QEmHqHpeoZM

```text
ssh -p 2221 admin@localhost
```

Cancel zerotouch

```text
localhost login:admin
localhost>zerotouch cancel
```

Configure IP for management

```text
config
interface ma1
ip address dhcp
dhcp client accept default-route
end

show run int ma1
show int ma1
```

Configure user with password

```text
en
config
username admin secret x245
```

Enable the eosAPI

```text
login> admin
> en
Switch# configure terminal
  Switch(config)# management api http-commands
  Switch(config-mgmt-api-http-cmds)# no shutdown
  Switch(config-mgmt-api-http-cmds)# protocol ?
    http         Configure HTTP server options
    https        Configure HTTPS server options
    unix-socket  Configure Unix Domain Socket
  Switch(config-mgmt-api-http-cmds)# protocol http
  Switch(config-mgmt-api-http-cmds)# end

  Switch# show management api http-commands
  Enabled:            Yes
  HTTPS server:       running, set to use port 443
  HTTP server:        running, set to use port 80
  Local HTTP server:  shutdown, no authentication, set to use port 8080
  Unix Socket server: shutdown, no authentication
```

Enable password:

```text
en
config
enable password xyrt1
// deletes the enable password
no enable password
```

see https://www.arista.com/en/um-eos/eos-user-security

Enter bash mode

```text
switch#conf t
switch(config)#bash sudo su -
```

understanding eapi
https://eos.arista.com/arista-eapi-101/

expose api (requires port forwarding)
<!-- markdown-link-check-disable -->
http://localhost:8080/explorer.html
<!-- markdown-link-check-enable -->

arista user manual
https://www.arista.com/en/um-eos

vagrant setup (not tested)
https://github.com/jerearista/vagrant-veos