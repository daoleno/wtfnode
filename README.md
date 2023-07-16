# WTFNode

WTFNode is a proxy for distributing requests to multiple evm node providers.

## Features

- Round-robin Load Balancing
- Rate Limiting
- Request Retry
- Failover

## Installation

```sh
go install github.com/daoleno/wtfnode-cli@latest
```

## Usage

```sh
mv example.config.toml config.toml
wtfnode-cli -c config.toml
```

## License

[MIT](https://choosealicense.com/licenses/mit/)
