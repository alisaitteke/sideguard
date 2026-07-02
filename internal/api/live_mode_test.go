//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/approvalmode"
)

func main() {
	ctx := context.Background()
	c := api.NewClient()
	m, err := c.GetApprovalMode(ctx)
	fmt.Printf("GET: mode=%s err=%v\n", m, err)
	err = c.SetApprovalMode(ctx, approvalmode.Auto)
	fmt.Printf("SET auto: err=%v\n", err)
	m, err = c.GetApprovalMode(ctx)
	fmt.Printf("GET after: mode=%s err=%v\n", m, err)
	os.Exit(0)
}
