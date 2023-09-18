# 👺 WTFNode

WTFNode is a proxy for distributing requests to multiple evm node providers.

## Features

- Round-robin Load Balancing
- Rate Limiting
- Request Retry
- Failover
- Batch Request
- Custom Method Mapping

## Installation

```sh
go install github.com/daoleno/wtfnode@latest
```

## Usage

```sh
mv example.proxy.toml proxy.toml
wtfnode proxy -c proxy.toml
```

## License

[MIT](https://choosealicense.com/licenses/mit/)
