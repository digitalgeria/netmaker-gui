# netmaker-gui
A responsive alternative UI for netmaker (https://github.com/gravitl/netmaker)


Built with go and html/templates.
Missing following features compared to netmaker-ui (https://github.com/gravitl/netmaker-ui)
- Oauth Suppport

- Only Supports CLIENT_MODE=off
  - creating new networks will fail if CLIENT_MODE is "on" or "contained"


You can use netmaker-gui at the same time as netmaker-ui. For example, one one running as dashboard.netmaker.example.com and the other at control.netmaker.example.com

![both](https://github.com/mattkasun/netmaker-gui/raw/develop/screenshots/netmaker-gui-ui.png "GUI and UI")


## Installation:

To use along side of your existing netmaker installation insert the following to your docker-compose.yml file

```
netmaker-gui
  container-name: netmaker-gui
  image: nusak/netmaker-gui:v0.6.3
  restart: unless-stopped
  ports:
    - "8080:8080"
  environment:
    DATABASE: sqlite
    CLIENT_MODE: "off"
  volumes:
    - dnsconfig:/config/dnsconfig
    - sqldata:/data
```

and add an appropriate entry to your proxy relay.

## Screenshots
### Browser
![netmaker-gui with browser](https://github.com/mattkasun/netmaker-gui/raw/develop/screenshots/netmaker-gui-browser.png "Netmaker-GUI with Browser")

### Mobile
![netmaker-gui with phone](https://github.com/mattkasun/netmaker-gui/raw/develop/screenshots/netmaker-gui-phone.png "Netmaker-GUI with Phone")

## Third Party Tools
- CSS - W3Schools https://w3.schools.com/w3css
- Icons - Material Icons https://fonts.google.com/icons
- Netmaker https://github.com/gravitl/netmaker
