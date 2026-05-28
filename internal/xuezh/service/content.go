package service

import "github.com/joshp123/xuezh/internal/xuezh/content"

type PutContentOptions struct {
	ContentType string
	Key         string
	Filename    string
	Data        []byte
}

func (App) PutContent(opts PutContentOptions) (content.ContentResult, error) {
	return content.PutContentBytes(opts.ContentType, opts.Key, opts.Filename, opts.Data)
}

func (App) GetContent(contentType, key string) (content.ContentResult, error) {
	return content.GetContent(contentType, key)
}

func (App) GetContentBytes(contentType, key string) (content.ContentResult, []byte, error) {
	return content.GetContentBytes(contentType, key)
}
