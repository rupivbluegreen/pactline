// Package document is the Go-side client for the document sidecar.
package document

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rupivbluegreen/pactline/internal/document/documentgrpc"
)

type Client struct {
	conn *grpc.ClientConn
	cli  documentgrpc.DocumentClient
}

func Dial(ctx context.Context, addr string) (*Client, error) {
	_ = ctx // grpc.NewClient is non-blocking; ctx kept for symmetry/future
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial document sidecar: %w", err)
	}
	return &Client{conn: conn, cli: documentgrpc.NewDocumentClient(conn)}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) Health(ctx context.Context) (string, error) {
	resp, err := c.cli.Health(ctx, &documentgrpc.HealthRequest{})
	if err != nil {
		return "", err
	}
	return resp.GetStatus(), nil
}

// Parse hands the document bytes to the sidecar and returns the response
// verbatim. The activity layer (internal/workflow/activities) projects this
// into domain types.
func (c *Client) Parse(ctx context.Context, documentID string, content []byte, mimeType string) (*documentgrpc.ParseResponse, error) {
	return c.cli.Parse(ctx, &documentgrpc.ParseRequest{
		DocumentId: documentID,
		Content:    content,
		MimeType:   mimeType,
	})
}
