package ocr

import (
	"context"
	"fmt"
	"log"
	"time"

	"SebasXeon/Fileoteca/internal/ocr/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type OcrClient struct {
	conn   *grpc.ClientConn
	client proto.OCREngineClient
	addr   string
}

func NewOcrClient(addr string) (*OcrClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OCR server at %s: %w", addr, err)
	}

	client := proto.NewOCREngineClient(conn)
	log.Printf("connected to OCR server at %s", addr)
	_ = ctx

	return &OcrClient{
		conn:   conn,
		client: client,
		addr:   addr,
	}, nil
}

func (c *OcrClient) ExtractText(ctx context.Context, id, filePath, fileType string) (string, error) {
	req := &proto.ExtractRequest{
		Id:       id,
		FilePath: filePath,
		FileType: fileType,
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	resp, err := c.client.ExtractText(ctx, req)
	if err != nil {
		return "", fmt.Errorf("OCR ExtractText failed for %s: %w", id, err)
	}

	return resp.Text, nil
}

func (c *OcrClient) GenerateThumbnail(ctx context.Context, id, filePath, fileType string) (string, error) {
	req := &proto.ThumbnailRequest{
		Id:       id,
		FilePath: filePath,
		FileType: fileType,
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	resp, err := c.client.GenerateThumbnail(ctx, req)
	if err != nil {
		return "", fmt.Errorf("GenerateThumbnail failed for %s: %w", id, err)
	}

	return resp.ThumbnailPath, nil
}

func (c *OcrClient) Close() error {
	return c.conn.Close()
}
