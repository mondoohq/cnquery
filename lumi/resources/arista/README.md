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

```
ssh -p 2221 admin@localhost
```

Enable the eosAPI

```
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

understanding eapi
https://eos.arista.com/arista-eapi-101/

expose api (requires port forwarding)
http://localhost:8080/explorer.html

arista user manual
https://www.arista.com/assets/data/pdf/user-manual/um-books/EOS-4.24.1F-Manual.pdf

vagrant setup (not tested)
https://github.com/jerearista/vagrant-veos