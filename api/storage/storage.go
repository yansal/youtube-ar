package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// New returns a new storage.
func New(bucket string) (*Storage, error) {
	s, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	return &Storage{
		bucket: bucket,
		s3:     s3.New(s),
	}, nil
}

// Storage is a storage.
type Storage struct {
	bucket string
	s3     *s3.S3
}

// Save saves file located at path.
func (s *Storage) Save(ctx context.Context, path string) (string, error) {
	// TODO: add logs

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	input := &s3.PutObjectInput{
		Body:   f,
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filepath.Base(path)),
	}

	switch {
	case strings.HasSuffix(path, ".mp3"):
		input.ContentType = aws.String("audio/mpeg")
	case strings.HasSuffix(path, ".mp4"):
		input.ContentType = aws.String("video/mp4")
	case strings.HasSuffix(path, ".webm"):
		input.ContentType = aws.String("video/webm")
	}

	if _, err := s.s3.PutObjectWithContext(ctx, input); err != nil {
		return "", err
	}
	return path, nil
}
