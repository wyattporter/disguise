# `disguise`

An stripped-down [`atmos/camo`](https://github.com/atmos/camo) alternative.
Provides endpoint `/:digest/:url`.


## example configuration
### `nginx`

    # /etc/nginx/sites-enabled/disguise
    # disguise 
    upstream disguise {
     server unix:/run/disguise/sock;
    }

    server {
     # ...

     # storage path for downloaded images
     set $disguise_store "/var/cache/nginx/disguise"
     # image proxy location
     location /i/ {
      rewrite ^/i/(.*)$ /$1 break;
      root $disguise_store;
      # check if downloaded, otherwise use disguise
      try_files $uri @disguise;
     }
     location @disguise {
      proxy_pass http://disguise;
      proxy_http_version 1.1;
      proxy_set_header Upgrade $http_upgrade;
      proxy_set_header Connection "upgrade";
      # save downloaded content in $disguise_store as file $uri
      proxy_store $disguise_store/$uri;
     }
    }

### `systemd`

    # /etc/systemd/system/disguise.service
    [Unit]
    Description=disguise: camo, simplified
    Requires=network.target
    After=network.target
    
    [Service]
    Type=simple
    Environment=CAMO_KEY=0x24FEEDFACEDEADBEEFCAFE
    Environment=NETWORK=unix
    Environment=ADDRESS=/run/disguise/sock
    ExecStartPre=/usr/bin/install --directory --owner=www-data --group=www-data /run/disguise
    ExecStart=/usr/bin/disguise -n "${NETWORK}" -a "${ADDRESS}"
    ExecStopPost=/bin/rm /run/disguise/sock
    User=www-data
    Group=www-data
    MemoryDenyWriteExecute=True
    PermissionsStartOnly=True
    PrivateDevices=True
    PrivateTmp=True
    ProtectHome=True
    ProtectKernelModules=True
    ProtectSystem=True
    
    [Install]
    WantedBy=multi-user.target
