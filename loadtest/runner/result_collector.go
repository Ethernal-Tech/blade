package runner

import (
	"context"
	"fmt"
)

// ResultCollector collects the results of the load test.
type ResultCollector struct {
	BalanceReadCountCh chan struct{}
	BalanceReadErrorCh chan error
	BalanceReadCount   int
	BalanceReadErrors  []error

	NonceReadCountCh chan struct{}
	NonceReadErrorCh chan error
	NonceReadCount   int
	NonceReadErrors  []error

	TxPoolStatusReadCountCh chan struct{}
	TxPoolStatusReadErrorCh chan error
	TxPoolStatusReadCount   int
	TxPoolStatusReadErrors  []error
}

// NewResultCollector creates a new ResultCollector instance.
func NewResultCollector() *ResultCollector {
	return &ResultCollector{
		BalanceReadCountCh:      make(chan struct{}, 3000),
		BalanceReadErrorCh:      make(chan error, 3000),
		NonceReadCountCh:        make(chan struct{}, 3000),
		NonceReadErrorCh:        make(chan error, 3000),
		TxPoolStatusReadCountCh: make(chan struct{}, 3000),
		TxPoolStatusReadErrorCh: make(chan error, 3000),
	}
}

// CollectResults collects the results of the load test.
func (r *ResultCollector) CollectResults(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.BalanceReadCountCh:
			r.BalanceReadCount++
		case err := <-r.BalanceReadErrorCh:
			r.BalanceReadErrors = append(r.BalanceReadErrors, err)
		case <-r.NonceReadCountCh:
			r.NonceReadCount++
		case err := <-r.NonceReadErrorCh:
			r.NonceReadErrors = append(r.NonceReadErrors, err)
		case <-r.TxPoolStatusReadCountCh:
			r.TxPoolStatusReadCount++
		case err := <-r.TxPoolStatusReadErrorCh:
			r.TxPoolStatusReadErrors = append(r.TxPoolStatusReadErrors, err)
		}
	}
}

// PrintResults prints the results of the load test.
func (r *ResultCollector) PrintResults() {
	fmt.Println("Total balance read count:", r.BalanceReadCount)
	fmt.Println("Total nonce read count:", r.NonceReadCount)
	fmt.Println("Total tx pool status read count:", r.TxPoolStatusReadCount)
	if len(r.BalanceReadErrors) > 0 {
		fmt.Println("====================================")
		fmt.Println("Balance read errors:")
		for i, err := range r.BalanceReadErrors {
			fmt.Printf("%d: %v\n", i, err)
		}
	}

	if len(r.NonceReadErrors) > 0 {
		fmt.Println("====================================")
		fmt.Println("Nonce read errors:")
		for i, err := range r.NonceReadErrors {
			fmt.Printf("%d: %v\n", i, err)
		}
	}

	if len(r.TxPoolStatusReadErrors) > 0 {
		fmt.Println("====================================")
		fmt.Println("Tx pool status read errors:")
		for i, err := range r.TxPoolStatusReadErrors {
			fmt.Printf("%d: %v\n", i, err)
		}
	}
}
