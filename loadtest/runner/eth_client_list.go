package runner

import "github.com/0xPolygon/polygon-edge/jsonrpc"

// ethClientList is a list of EthClients
type ethClientList []*jsonrpc.EthClient

// newEthClientList creates a new list of EthClients from the given JSON-RPC URLs
func newEthClientList(jsonRPCURLs []string) (ethClientList, error) {
	clients := make(ethClientList, 0, len(jsonRPCURLs))
	for _, url := range jsonRPCURLs {
		client, err := jsonrpc.NewEthClient(url)
		if err != nil {
			return nil, err
		}

		clients = append(clients, client)
	}

	return clients, nil
}

// close closes all the EthClients in the list
func (ecl ethClientList) close() error {
	for _, client := range ecl {
		if err := client.Close(); err != nil {
			return err
		}
	}

	return nil
}

// getClient returns an EthClient from the list of clients for the given account index
func (ecl ethClientList) getClientForAccount(accountIndex int) *jsonrpc.EthClient {
	return ecl[accountIndex%len(ecl)]
}

func (ecl ethClientList) getClient() *jsonrpc.EthClient {
	return ecl[0]
}
