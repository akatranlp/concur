# Concur

Concur is a go version of the npm package concurrently. It allows you to run multiple commands concurrently.
It also adds the functionality to run commands before and after the main commands, for infra startup and cleanup.

Secondly it adds the functionality for healthchecks, which are always at the bottom of all logs to ensure that the infra and app is up and running.

## Installation

### via curl

```bash
curl -fL https://github.com/akatranlp/concur/releases/download/v0.1.0/concur_Linux_x86_64.tar.gz | tar -xz
sudo mv concur /usr/local/bin/.
```

### via go

```bash
go install github.com/akatranlp/concur@latest
```

