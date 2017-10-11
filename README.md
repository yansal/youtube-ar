# youtube-ar

## Requirements

* `tor` and `youtube-dl` in `PATH`
* A PostgreSQL database
* An S3 bucket

## Migrate schema

    createdb youtube-ar && psql youtube-ar < schema.sql

## Start application

    go install && AWS_REGION=<aws-region> S3_BUCKET=<s3-bucket> youtube-ar

## TODO(features)

* plug webhooks from youtube, soundcloud, etc.
* allow to directly download video
* more storage services
