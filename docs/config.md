# Configration Format
**NOTE** The format is unstable, it may be changed in the future version.
The `config.toml` is the runtime configuration for rsync-os (both daemon/server and client). Similiar to `rsyncd.conf`, it can control authentication, access, logging, available modules and which storage backend rsync-os uses(local file storage, s3 object storage). 

A storage backend begins with the name of the baackend in the square brackets and continues until the next square brackets. 

* To configure a s3 object storage:
```toml
[minio]
  endpoint = "192.168.11.192:9000"  # Require
  keyAccess = "minioadmin"          # Require
  keySecret = "minioadmin"          # Require
  mods = []   # Unlike rsync, all module will specify in the storage backend.
````

