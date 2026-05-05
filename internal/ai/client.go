// Package ai is the Go-side client for the AI sidecar.
package ai

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rupivbluegreen/pactline/internal/ai/aigrpc"
	"github.com/rupivbluegreen/pactline/internal/document/documentgrpc"
)

type Client struct {
	conn *grpc.ClientConn
	cli  aigrpc.AIClient
}

func Dial(ctx context.Context, addr string) (*Client, error) {
	_ = ctx // grpc.NewClient is non-blocking; ctx kept for symmetry/future
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial ai sidecar: %w", err)
	}
	return &Client{conn: conn, cli: aigrpc.NewAIClient(conn)}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) Health(ctx context.Context) (string, error) {
	resp, err := c.cli.Health(ctx, &aigrpc.HealthRequest{})
	if err != nil {
		return "", err
	}
	return resp.GetStatus(), nil
}

// ExtractFields forwards the request to the sidecar. The sidecar enforces
// citation completeness and returns FAILED_PRECONDITION when violated; the
// activity layer maps that to a workflow-visible error.
func (c *Client) ExtractFields(
	ctx context.Context,
	documentID, parsedText string,
	segments []*documentgrpc.TextSegment,
	fieldNames []string,
	modelID, promptVersion string,
) (*aigrpc.ExtractResponse, error) {
	return c.cli.ExtractFields(ctx, &aigrpc.ExtractRequest{
		DocumentId:    documentID,
		ParsedText:    parsedText,
		Segments:      segments,
		FieldNames:    fieldNames,
		ModelId:       modelID,
		PromptVersion: promptVersion,
	})
}
