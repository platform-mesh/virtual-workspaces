# Platform-Mesh virtual workspaces


## Getting Started

To start the service locally you need some trusted certificates. You can generate them using the following command:

```bash
mkcert -cert-file=.secret/apiserver.crt -key-file=.secret/apiserver.key localhost
```

Also make sure you have `mkcert` installed and trusted its store in your system.

The Kubeconfigs server url needs to point to the **root** of the kcp instance (*not the root cluster*), meaning only the url.

### VSCode Debug config:

```json
{
    // Use IntelliSense to learn about possible attributes.
    // Hover to view descriptions of existing attributes.
    // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Package",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/main.go",
            "args": [
                "start",
                "--tls-cert-file=${workspaceFolder}/.secret/apiserver.crt",
                "--tls-private-key-file=${workspaceFolder}/.secret/apiserver.key",
                "--secure-port=6443",
                "--bind-address=0.0.0.0"
            ],
            "envFile": "${workspaceFolder}/.env",
        }
    ]
}
```
