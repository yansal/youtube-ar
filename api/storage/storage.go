package storage

import (
	"context"
	"io"
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
func (s *Storage) Save(ctx context.Context, path string, reader io.ReadSeeker) error {
	// TODO: add logs

	input := &s3.PutObjectInput{
		Body:   reader,
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}

	switch {
	case strings.HasSuffix(path, ".mp3"):
		input.ContentType = aws.String("audio/mpeg")
	case strings.HasSuffix(path, ".mp4"):
		input.ContentType = aws.String("video/mp4")
	case strings.HasSuffix(path, ".webm"):
		input.ContentType = aws.String("video/webm")
	}

	_, err := s.s3.PutObjectWithContext(ctx, input)
	return err
}
