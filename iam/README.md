# S3 mock

```
sudo pacman -Sy s3cmd
```

```~/.s3cfg
host_bucket = localhost:9090/%(bucket)s
host_domain = localhost:9090
use_https = false
```

```
s3cmd --host=localhost:9090 ls
```
