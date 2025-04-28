package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func submitDailyReport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := request.Params.Arguments["name"].(string)
	if !ok {
		return nil, errors.New("name must be a string")
	}

	return mcp.NewToolResultText(fmt.Sprintf("Hello, %s!", name)), nil
}

func main() {
	s := server.NewMCPServer(
		"times-esa-mcp-server",
		"1.0.0",
	)

	tool := mcp.NewTool("times-esa",
		mcp.WithDescription("times-esaに日報を投稿します"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the person to greet"),
		),
	)

	s.AddTool(tool, submitDailyReport)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
