package receiver

import "context"

type DataReceiver interface {
	GetFile(ctx context.Context, fileID string) ([]byte, error)
	SaveFile(ctx context.Context, fileID string, data []byte) error
}
