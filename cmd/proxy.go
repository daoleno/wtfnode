package cmd

import (
	"github.com/daoleno/wtfnode/config"
	"github.com/daoleno/wtfnode/proxy"
	"github.com/spf13/cobra"
)

var proxyConfPath string

// proxyCmd represents the proxy command
var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "A proxy for distributing requests to multiple evm node providers.",
	Long: `Supported features:
				- Round-robin Load Balancing
				- Rate Limiting
				- Request Retry
				- Failover
				- Batch Request
				- Custom Method Mapping`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load the mapping of JSON-RPC methods to backend URLs
		conf := config.LoadConfig(proxyConfPath)
		p := proxy.NewProxy(conf)
		p.Start()
	},
}

func init() {
	rootCmd.AddCommand(proxyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// proxyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	proxyCmd.Flags().StringVarP(&proxyConfPath, "config", "c", "proxy.toml", "Path to proxy config file")
}
