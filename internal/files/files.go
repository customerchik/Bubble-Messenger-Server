package files

import (
	"context"
	"fmt"
	"os"

	pb "bubble/internal/files/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	saveAvatar = "save_avatar"
	saveFile   = "save_file"
)

type Client struct {
	service pb.FilesServiceClient
	conn    *grpc.ClientConn
}

type FilesService interface {
	DownloadFileTempPath(ctx context.Context, filepath string) (string, error)
	SaveAvatarTempPath(ctx context.Context, ch chan<- string)
	SaveFileTempPath(ctx context.Context, ch chan<- string)
	DelFile(ctx context.Context, storagePath string) error
}

var usersServiceHost = os.Getenv("FILES_SERVICE_HOST_GRPC_PORT")

func NewClient() (*Client, error) {
	client := &Client{}
	if client.service != nil {
		return client, nil
	}

	conn, err := grpc.NewClient(usersServiceHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return client, err
	}

	client.service = pb.NewFilesServiceClient(conn)
	client.conn = conn
	return client, nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) DownloadFileTempPath(ctx context.Context, filepath string) (string, error) {
	stream, err := c.service.TempPath(ctx, &pb.TempPathReqeust{
		Info: filepath,
	})
	if err != nil {
		return "", err
	}
	defer stream.CloseSend()

	resp, err := stream.Recv()
	if err != nil {
		return "", err
	}

	return resp.TempPath, nil
}

func (c *Client) SaveAvatarTempPath(ctx context.Context, ch chan<- string) {
	c.tempPath(ctx, saveAvatar, ch)
}

func (c *Client) SaveFileTempPath(ctx context.Context, ch chan<- string) {
	c.tempPath(ctx, saveFile, ch)
}

func (c *Client) tempPath(ctx context.Context, filepath string, ch chan<- string) {
	stream, err := c.service.TempPath(ctx, &pb.TempPathReqeust{
		Info: filepath,
	})
	if err != nil {
		ch <- fmt.Sprintf("Error: %s", err)
		return
	}
	defer stream.CloseSend()

	resp, err := stream.Recv()
	if err != nil {
		ch <- fmt.Sprintf("Error: %s", err)
		return
	}
	ch <- resp.TempPath

	resp, err = stream.Recv()
	if err != nil {
		ch <- fmt.Sprintf("Error send recv: %s", err)
		return
	}
	if resp.StoragePath != "" {
		ch <- resp.StoragePath
		return
	} else {
		ch <- "Error"
		return
	}
}

func (c *Client) DelFile(ctx context.Context, storagePath string) error {
	_, err := c.service.Del(ctx, &pb.DelFileRequest{
		StoragePath: storagePath,
	})
	if err != nil {
		return err
	}

	return nil
}
